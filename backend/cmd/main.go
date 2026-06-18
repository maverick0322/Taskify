package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/maverick0322/taskify/backend/docs"
	"github.com/maverick0322/taskify/backend/internal/adapters/auth"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/adapters/logging"
	"github.com/maverick0322/taskify/backend/internal/adapters/repositories"
	adapterutil "github.com/maverick0322/taskify/backend/internal/adapters/util"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	serverReadHeaderTimeout = 5 * time.Second
	serverShutdownTimeout   = 10 * time.Second
	corsAllowedMethods      = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders      = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"
)

var defaultCORSAllowedOrigins = []string{
	"http://localhost:1420",
	"http://localhost:5173",
	"http://localhost:8080",
	"http://tauri.localhost",
	"https://tauri.localhost",
}

// @title Taskify API
// @version 1.0
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if err := run(); err != nil {
		log.Fatalf("application stopped: %v", err)
	}
}

func run() error {
	if err := loadLocalEnvironment(); err != nil {
		return err
	}

	config, err := loadAppConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("invalid application configuration: %w", err)
	}

	baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	applicationLogger := logging.NewSlogLogger(baseLogger)

	startupContext, cancelStartup := context.WithTimeout(context.Background(), sqliteStartupTimeout)
	defer cancelStartup()

	isProduction := config.isProduction()
	var sqliteDatabase *sql.DB
	var remoteDatabase *sql.DB
	var remotePool *pgxpool.Pool

	if isProduction && config.remoteDatabaseURL == "" {
		return fmt.Errorf("REMOTE_DB_URL is required when ENV=production")
	}

	if isProduction {
		remoteDatabase, err = openRemotePostgresDatabase(startupContext, config.remoteDatabaseURL)
		if err != nil {
			return err
		}
		defer remoteDatabase.Close()

		remotePool, err = openRemotePostgresPool(startupContext, config.remoteDatabaseURL)
		if err != nil {
			return err
		}
		defer remotePool.Close()
	} else {
		sqliteDatabase, err = openLocalSQLiteDatabase(startupContext)
		if err != nil {
			return err
		}
		defer sqliteDatabase.Close()

		if config.remoteDatabaseURL != "" {
			remoteDatabase, err = openRemotePostgresDatabase(startupContext, config.remoteDatabaseURL)
			if err != nil {
				applicationLogger.Warn("remote sync disabled because postgres initialization failed", "error", err)
			} else {
				defer remoteDatabase.Close()
			}
		}
	}
	passwordHasher, err := auth.NewBcryptHasher(config.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to initialize password hasher: %w", err)
	}

	tokenGenerator, err := auth.NewJWTTokenGenerator(config.jwtSecret, config.accessTokenTTL, config.refreshTokenTTL)
	if err != nil {
		return fmt.Errorf("failed to initialize token generator: %w", err)
	}
	tokenValidator, ok := tokenGenerator.(ports.TokenValidator)
	if !ok {
		return fmt.Errorf("failed to initialize token validator")
	}

	idGenerator := adapterutil.NewUUIDGenerator()
	var userRepository ports.UserRepository
	var sessionRepository ports.SessionRepository
	var taskRepository ports.TaskRepository
	var boardRepository ports.BoardRepository
	var columnRepository ports.ColumnRepository
	var transactionRepository ports.TransactionRepository
	var creditCardRepository ports.CreditCardRepository
	var financialAccountRepository ports.FinancialAccountRepository
	var notificationRepository ports.NotificationRepository

	if isProduction {
		userRepository = repositories.NewPostgresUserRepository(remotePool, applicationLogger)
		sessionRepository = repositories.NewPostgresSessionRepository(remotePool, applicationLogger)
		taskRepository = repositories.NewPostgresTaskRepository(remotePool, applicationLogger)
		boardRepository = repositories.NewPostgresBoardRepository(remotePool, applicationLogger)
		columnRepository = repositories.NewPostgresColumnRepository(remotePool, applicationLogger)
		transactionRepository = repositories.NewPostgresTransactionRepository(remotePool, applicationLogger)
		creditCardRepository = repositories.NewPostgresCreditCardRepository(remotePool, applicationLogger)
		financialAccountRepository = repositories.NewPostgresFinancialAccountRepository(remotePool, applicationLogger)
		notificationRepository = repositories.NewPostgresNotificationRepository(remoteDatabase, applicationLogger)
	} else {
		userRepository = repositories.NewSQLiteUserRepository(sqliteDatabase, applicationLogger)
		sessionRepository = repositories.NewSQLiteSessionRepository(sqliteDatabase, applicationLogger)
		taskRepository = repositories.NewSQLiteTaskRepository(sqliteDatabase, applicationLogger)
		boardRepository = repositories.NewSQLiteBoardRepository(sqliteDatabase, applicationLogger)
		columnRepository = repositories.NewSQLiteColumnRepository(sqliteDatabase, applicationLogger)
		transactionRepository = repositories.NewSQLiteTransactionRepository(sqliteDatabase, applicationLogger)
		creditCardRepository = repositories.NewSQLiteCreditCardRepository(sqliteDatabase, applicationLogger)
		financialAccountRepository = repositories.NewSQLiteFinancialAccountRepository(sqliteDatabase, applicationLogger)
		notificationRepository = repositories.NewSQLiteNotificationRepository(sqliteDatabase, applicationLogger)
	}
	userUseCase := services.NewUserService(userRepository, sessionRepository, passwordHasher, tokenGenerator, idGenerator, applicationLogger)
	taskUseCase := services.NewTaskService(taskRepository, boardRepository, columnRepository, idGenerator, applicationLogger)
	boardUseCase := services.NewBoardService(boardRepository, columnRepository, idGenerator, applicationLogger)
	transactionUseCase := services.NewTransactionService(transactionRepository, idGenerator, applicationLogger, financialAccountRepository)
	creditCardUseCase := services.NewCreditCardService(creditCardRepository, transactionRepository, idGenerator, applicationLogger, financialAccountRepository)
	financialAccountUseCase := services.NewFinancialAccountService(financialAccountRepository, idGenerator, applicationLogger, transactionRepository)
	notificationUseCase := services.NewNotificationService(notificationRepository, applicationLogger)
	var syncService *services.SyncService
	if !isProduction && remoteDatabase != nil {
		syncService = services.NewSyncService(sqliteDatabase, remoteDatabase, services.SyncDialectPostgres, applicationLogger)
	}
	userHandler := handlers.NewUserHandler(userUseCase, applicationLogger)
	taskHandler := handlers.NewTaskHandler(taskUseCase, applicationLogger)
	boardHandler := handlers.NewBoardHandler(boardUseCase, applicationLogger)
	transactionHandler := handlers.NewTransactionHandler(transactionUseCase, applicationLogger, notificationUseCase)
	creditCardHandler := handlers.NewCreditCardHandler(creditCardUseCase, applicationLogger)
	financialAccountHandler := handlers.NewFinancialAccountHandler(financialAccountUseCase, applicationLogger)
	notificationHandler := handlers.NewNotificationHandler(notificationUseCase, applicationLogger)
	systemHandler := handlers.NewSystemHandler(sqliteDatabase, syncService, applicationLogger)
	authMiddleware := middleware.NewAuthMiddleware(tokenValidator, applicationLogger)

	router := chi.NewRouter()
	router.Use(withCORS(config.corsAllowedOrigins))
	router.Get("/swagger/*", httpSwagger.WrapHandler)
	userHandler.RegisterRoutes(router)
	router.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(authMiddleware.RequireAuthentication)
		userHandler.RegisterProtectedRoutes(protectedRouter)
		taskHandler.RegisterRoutes(protectedRouter)
		boardHandler.RegisterRoutes(protectedRouter)
		transactionHandler.RegisterRoutes(protectedRouter)
		creditCardHandler.RegisterRoutes(protectedRouter)
		financialAccountHandler.RegisterRoutes(protectedRouter)
		notificationHandler.RegisterRoutes(protectedRouter)
		systemHandler.RegisterRoutes(protectedRouter)
	})

	server := &http.Server{
		Addr:              ":" + config.port,
		Handler:           router,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	shutdownContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if !isProduction && remoteDatabase != nil {
		go startSyncWorker(shutdownContext, syncService, applicationLogger)
	}
	if !isProduction {
		go startAvatarStorageWorker(shutdownContext, sqliteDatabase, config.supabaseURL, config.supabaseServiceKey, applicationLogger)
	}
	go startNotificationWorker(shutdownContext, notificationUseCase, applicationLogger)

	serverErrors := make(chan error, 1)
	go startHTTPServer(server, serverErrors)

	applicationLogger.Info("http server started", "port", config.port)
	select {
	case <-shutdownContext.Done():
	case err := <-serverErrors:
		return fmt.Errorf("http server failed: %w", err)
	}
	stopSignals()

	// HTTP shutdown runs before closing SQLite so in-flight requests can finish their database work.
	gracefulShutdownContext, cancelGracefulShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancelGracefulShutdown()

	if err := server.Shutdown(gracefulShutdownContext); err != nil {
		return fmt.Errorf("failed to gracefully shutdown http server: %w", err)
	}

	applicationLogger.Info("http server stopped")
	return nil
}

func loadLocalEnvironment() error {
	envPaths := []string{
		".env",
		"../.env",
		"../../backend/.env",
		"../backend/.env",
	}

	for _, envPath := range envPaths {
		if err := godotenv.Overload(envPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to load local environment file %s: %w", envPath, err)
		}
		return nil
	}

	return nil
}

func startHTTPServer(server *http.Server, serverErrors chan<- error) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serverErrors <- err
	}
}

func withCORS(configuredOrigins []string) func(http.Handler) http.Handler {
	allowedOrigins := append([]string{}, defaultCORSAllowedOrigins...)
	allowedOrigins = append(allowedOrigins, configuredOrigins...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			origin := r.Header.Get("Origin")
			headers.Add("Vary", "Origin")

			if isAllowedCORSOrigin(origin, allowedOrigins) {
				headers.Set("Access-Control-Allow-Origin", origin)
				headers.Set("Access-Control-Allow-Methods", corsAllowedMethods)
				headers.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			}

			if r.Method == http.MethodOptions {
				if origin != "" && !isAllowedCORSOrigin(origin, allowedOrigins) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedCORSOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if strings.EqualFold(origin, allowedOrigin) {
			return true
		}
	}

	return strings.HasPrefix(origin, "https://") && strings.HasSuffix(origin, ".vercel.app")
}
