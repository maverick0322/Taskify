package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	postgresStartupTimeout  = 5 * time.Second
	corsAllowedOrigin       = "*"
	corsAllowedMethods      = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders      = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"
)

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
	if err := godotenv.Overload(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load local environment file: %w", err)
	}

	config, err := loadAppConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("invalid application configuration: %w", err)
	}

	baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	applicationLogger := logging.NewSlogLogger(baseLogger)

	startupContext, cancelStartup := context.WithTimeout(context.Background(), postgresStartupTimeout)
	defer cancelStartup()

	postgresPool, err := pgxpool.New(startupContext, config.databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create postgres connection pool: %w", err)
	}
	defer postgresPool.Close()

	if err := postgresPool.Ping(startupContext); err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
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
	userRepository := repositories.NewPostgresUserRepository(postgresPool, applicationLogger)
	sessionRepository := repositories.NewPostgresSessionRepository(postgresPool, applicationLogger)
	taskRepository := repositories.NewPostgresTaskRepository(postgresPool, applicationLogger)
	boardRepository := repositories.NewPostgresBoardRepository(postgresPool, applicationLogger)
	columnRepository := repositories.NewPostgresColumnRepository(postgresPool, applicationLogger)
	userUseCase := services.NewUserService(userRepository, sessionRepository, passwordHasher, tokenGenerator, idGenerator, applicationLogger)
	taskUseCase := services.NewTaskService(taskRepository, boardRepository, idGenerator, applicationLogger)
	boardUseCase := services.NewBoardService(boardRepository, columnRepository, idGenerator, applicationLogger)
	userHandler := handlers.NewUserHandler(userUseCase, applicationLogger)
	taskHandler := handlers.NewTaskHandler(taskUseCase, applicationLogger)
	boardHandler := handlers.NewBoardHandler(boardUseCase, applicationLogger)
	authMiddleware := middleware.NewAuthMiddleware(tokenValidator, applicationLogger)

	router := chi.NewRouter()
	router.Use(withCORS)
	router.Get("/swagger/*", httpSwagger.WrapHandler)
	userHandler.RegisterRoutes(router)
	router.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(authMiddleware.RequireAuthentication)
		taskHandler.RegisterRoutes(protectedRouter)
		boardHandler.RegisterRoutes(protectedRouter)
	})

	server := &http.Server{
		Addr:              ":" + config.port,
		Handler:           router,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	shutdownContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serverErrors := make(chan error, 1)
	go startHTTPServer(server, serverErrors)

	applicationLogger.Info("http server started", "port", config.port)
	select {
	case <-shutdownContext.Done():
	case err := <-serverErrors:
		return fmt.Errorf("http server failed: %w", err)
	}
	stopSignals()

	// HTTP shutdown runs before closing Postgres so in-flight requests can finish their database work.
	gracefulShutdownContext, cancelGracefulShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancelGracefulShutdown()

	if err := server.Shutdown(gracefulShutdownContext); err != nil {
		return fmt.Errorf("failed to gracefully shutdown http server: %w", err)
	}

	applicationLogger.Info("http server stopped")
	return nil
}

func startHTTPServer(server *http.Server, serverErrors chan<- error) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serverErrors <- err
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Access-Control-Allow-Origin", corsAllowedOrigin)
		headers.Set("Access-Control-Allow-Methods", corsAllowedMethods)
		headers.Set("Access-Control-Allow-Headers", corsAllowedHeaders)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
