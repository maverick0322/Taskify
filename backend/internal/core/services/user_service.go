package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// Service-level errors for predictable flow control in the HTTP handlers.
var (
	ErrUserAlreadyExists     = errors.New("service: user with this email already exists")
	ErrInvalidCredentials    = errors.New("service: invalid email or password")
	ErrInvalidAvatarPath     = errors.New("service: avatar path cannot be empty")
	ErrInternalProcessing    = errors.New("service: an internal error occurred while processing the request")
	ErrInvalidRefreshToken   = errors.New("service: invalid refresh token")
	ErrSessionRevoked        = errors.New("service: refresh session has been revoked")
	ErrRefreshSessionExpired = errors.New("service: refresh session has expired")
	ErrRemoteAuthUnavailable = errors.New("service: remote auth is unavailable")
	ErrInvalidRemoteUserData = errors.New("service: remote user data is invalid")
)

type remoteAuthClient interface {
	AuthenticateRemoteSession(ctx context.Context, email, password string) (ports.TokenPair, error)
	RegisterRemoteUser(ctx context.Context, email, password, firstName, lastName, birthDate string) error
	GetRemoteProfile(ctx context.Context) (*RemoteUserProfile, error)
	SyncOnce(ctx context.Context) error
	ForceFullPull(ctx context.Context) error
	NeedsBootstrapPull(ctx context.Context) (bool, error)
}

// userService implements ports.UserUseCase.
// Unexported struct ensures it can only be created via the constructor (Factory Pattern).
type userService struct {
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	hasher      ports.PasswordHasher
	tokenGen    ports.TokenGenerator
	idGen       ports.IDGenerator
	logger      ports.Logger
	remoteAuth  remoteAuthClient
}

// NewUserService creates a new instance injecting all required dependencies.
func NewUserService(
	repo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	hasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	idGen ports.IDGenerator,
	logger ports.Logger,
) ports.UserUseCase {
	return NewUserServiceWithRemoteAuth(repo, sessionRepo, hasher, tokenGen, idGen, logger, nil)
}

func NewUserServiceWithRemoteAuth(
	repo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	hasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	idGen ports.IDGenerator,
	logger ports.Logger,
	remoteAuth remoteAuthClient,
) ports.UserUseCase {
	return &userService{
		userRepo:    repo,
		sessionRepo: sessionRepo,
		hasher:      hasher,
		tokenGen:    tokenGen,
		idGen:       idGen,
		logger:      logger,
		remoteAuth:  remoteAuth,
	}
}

func (s *userService) Register(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error) {
	if s.remoteAuth != nil {
		return s.registerRemote(ctx, email, plainPassword, firstName, lastName, birthDate)
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		// Log as Warn since it's a client error, not a system failure. Do not log the email to protect PII.
		s.logger.Warn("registration attempt with existing email")
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		s.logger.Error("failed to verify existing user during registration", "error", err)
		return nil, ErrInternalProcessing
	}

	hashedPassword, err := s.hasher.Hash(plainPassword)
	if err != nil {
		s.logger.Error("failed to hash password during registration", "error", err)
		return nil, ErrInternalProcessing
	}

	profile, err := domain.NewUserProfile(firstName, lastName, birthDate)
	if err != nil {
		return nil, err // Return domain error directly (e.g., ErrUnderageUser)
	}

	userID := s.idGen.Generate()
	newUser, err := domain.NewUser(userID, email, hashedPassword, profile)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.Save(ctx, newUser); err != nil {
		s.logger.Error("failed to save new user to database", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	s.logger.Info("user registered successfully", "userID", userID)
	return newUser, nil
}

func (s *userService) Authenticate(ctx context.Context, email, plainPassword string) (string, string, error) {
	if s.remoteAuth != nil {
		return s.authenticateDesktop(ctx, email, plainPassword)
	}

	return s.authenticateLocal(ctx, email, plainPassword)
}

func (s *userService) authenticateLocal(ctx context.Context, email, plainPassword string) (string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, ports.ErrUserNotFound) {
		s.logger.Warn("authentication failed: user not found")
		// We return a generic invalid credentials error to prevent email enumeration attacks (Security).
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		s.logger.Error("failed to retrieve user during authentication", "error", err)
		return "", "", ErrInternalProcessing
	}
	if user == nil {
		s.logger.Warn("authentication failed: user not found")
		return "", "", ErrInvalidCredentials
	}

	if err := s.hasher.Compare(plainPassword, user.PasswordHash()); err != nil {
		s.logger.Warn("authentication failed: incorrect password", "userID", user.ID())
		return "", "", ErrInvalidCredentials
	}

	return s.issueLocalSession(ctx, user)
}

func (s *userService) authenticateDesktop(ctx context.Context, email, plainPassword string) (string, string, error) {
	s.logger.Info("[AUTH][DESKTOP] Iniciando autenticación desktop", "email", email)
	localUser, localErr := s.userRepo.GetByEmail(ctx, email)
	if localErr != nil && !errors.Is(localErr, ports.ErrUserNotFound) {
		s.logger.Error("failed to retrieve local user during desktop authentication", "error", localErr)
		return "", "", ErrInternalProcessing
	}
	if localErr == nil && localUser != nil {
		s.logger.Info("[AUTH][DESKTOP] Usuario local encontrado; se intentará validar contra remoto primero", "email", email, "userID", localUser.ID())
	} else {
		s.logger.Info("[AUTH][DESKTOP] Usuario local no encontrado; se intentará hidratación remota", "email", email)
	}

	remoteUser, err := s.authenticateAndHydrateRemoteUser(ctx, email, plainPassword)
	if err == nil {
		if err := s.runInitialDesktopSync(ctx, email); err != nil {
			s.logger.Error("[AUTH][DESKTOP] El bootstrap inicial falló después del login remoto", "email", email, "error", err)
			return "", "", ErrInternalProcessing
		}
		s.logger.Info("[AUTH][DESKTOP] Autenticación remota e hidratación local completadas", "email", email, "userID", remoteUser.ID())
		return s.issueLocalSession(ctx, remoteUser)
	}
	if errors.Is(err, ErrInvalidCredentials) {
		s.logger.Warn("[AUTH][DESKTOP] Login remoto rechazado por credenciales inválidas", "email", email)
		return "", "", ErrInvalidCredentials
	}
	if localErr == nil && localUser != nil && errors.Is(err, ErrRemoteAuthUnavailable) {
		s.logger.Warn("remote auth unavailable, falling back to local cached credentials", "email", email)
		if compareErr := s.hasher.Compare(plainPassword, localUser.PasswordHash()); compareErr != nil {
			s.logger.Warn("[AUTH][DESKTOP] Fallback local rechazado por hash local inválido", "email", email, "userID", localUser.ID())
			return "", "", ErrInvalidCredentials
		}
		s.logger.Info("[AUTH][DESKTOP] Fallback local exitoso", "email", email, "userID", localUser.ID())
		return s.issueLocalSession(ctx, localUser)
	}
	if errors.Is(err, ErrRemoteAuthUnavailable) {
		s.logger.Warn("[AUTH][DESKTOP] Login desktop sin red o auth remota no disponible", "email", email)
		return "", "", ErrRemoteAuthUnavailable
	}
	s.logger.Error("[AUTH][DESKTOP] Login desktop falló en hidratación o sesión local", "email", email, "error", err)
	return "", "", ErrInternalProcessing
}

func (s *userService) registerRemote(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error) {
	if err := s.remoteAuth.RegisterRemoteUser(ctx, email, plainPassword, firstName, lastName, birthDate.Format("2006-01-02")); err != nil {
		return nil, err
	}

	user, err := s.authenticateAndHydrateRemoteUser(ctx, email, plainPassword)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) runInitialDesktopSync(ctx context.Context, email string) error {
	if s.remoteAuth == nil {
		return nil
	}

	needsBootstrap, err := s.remoteAuth.NeedsBootstrapPull(ctx)
	if err != nil {
		s.logger.Error("[AUTH][DESKTOP] No se pudo determinar el modo de sync inicial", "email", email, "error", err)
		return err
	}
	if needsBootstrap {
		s.logger.Info("[AUTH][DESKTOP] Ejecutando ForceFullPull inicial desde el login principal", "email", email)
		if err := s.remoteAuth.ForceFullPull(ctx); err != nil {
			return err
		}
		s.logger.Info("[AUTH][DESKTOP] ForceFullPull inicial completado desde el login principal", "email", email)
		return nil
	}

	s.logger.Info("[AUTH][DESKTOP] Ejecutando SyncOnce inicial desde el login principal", "email", email)
	if err := s.remoteAuth.SyncOnce(ctx); err != nil {
		return err
	}
	s.logger.Info("[AUTH][DESKTOP] SyncOnce inicial completado desde el login principal", "email", email)
	return nil
}

func (s *userService) authenticateAndHydrateRemoteUser(ctx context.Context, email, plainPassword string) (*domain.User, error) {
	s.logger.Info("[AUTH][DESKTOP] Intentando autenticar contra Render", "email", email)
	if _, err := s.remoteAuth.AuthenticateRemoteSession(ctx, email, plainPassword); err != nil {
		s.logger.Warn("[AUTH][DESKTOP] Render rechazó o no pudo completar el login", "email", email, "error", err)
		return nil, err
	}
	s.logger.Info("[AUTH][DESKTOP] Login remoto exitoso; solicitando perfil remoto", "email", email)

	remoteProfile, err := s.remoteAuth.GetRemoteProfile(ctx)
	if err != nil {
		if errors.Is(err, ErrRemoteAuthUnavailable) {
			s.logger.Warn("[AUTH][DESKTOP] No se pudo leer /users/me después del login remoto", "email", email, "error", err)
			return nil, err
		}
		s.logger.Error("failed to fetch remote profile during desktop authentication", "error", err)
		return nil, ErrInternalProcessing
	}
	s.logger.Info("[AUTH][DESKTOP] Perfil remoto recibido", "email", remoteProfile.Email, "remoteUserID", remoteProfile.ID)

	localUser, localErr := s.userRepo.GetByEmail(ctx, remoteProfile.Email)
	if localErr != nil && !errors.Is(localErr, ports.ErrUserNotFound) {
		s.logger.Error("failed to resolve local user during remote hydration", "error", localErr)
		return nil, ErrInternalProcessing
	}
	if localErr == nil && localUser != nil {
		s.logger.Info("[AUTH][DESKTOP] Se actualizará usuario local existente con datos remotos", "email", remoteProfile.Email, "localUserID", localUser.ID())
	} else {
		s.logger.Info("[AUTH][DESKTOP] Se creará usuario local desde perfil remoto", "email", remoteProfile.Email, "remoteUserID", remoteProfile.ID)
	}

	normalizeRemoteProfile(remoteProfile, localUser, email)
	s.logger.Info("[AUTH][DESKTOP] Perfil remoto normalizado para hidratación local", "email", remoteProfile.Email, "firstName", remoteProfile.FirstName, "lastName", remoteProfile.LastName, "birthDateZero", remoteProfile.BirthDate.IsZero())

	hashedPassword, err := s.hasher.Hash(plainPassword)
	if err != nil {
		s.logger.Error("failed to hash local password during remote hydration", "error", err)
		return nil, ErrInternalProcessing
	}

	profile, err := domain.NewUserProfile(remoteProfile.FirstName, remoteProfile.LastName, remoteProfile.BirthDate)
	if err != nil {
		s.logger.Error("remote profile is invalid for local hydration", "error", err)
		return nil, ErrInternalProcessing
	}

	userID := remoteProfile.ID
	avatarLocalPath := remoteProfile.AvatarLocalPath
	if localErr == nil && localUser != nil {
		userID = localUser.ID()
		if avatarLocalPath == "" {
			avatarLocalPath = localUser.AvatarLocalPath()
		}
	}

	user, err := domain.NewUser(userID, remoteProfile.Email, hashedPassword, profile)
	if err != nil {
		s.logger.Error("failed to build hydrated local user", "error", err)
		return nil, ErrInternalProcessing
	}
	user.UpdateAvatar(avatarLocalPath, remoteProfile.AvatarURL)

	if err := s.userRepo.Upsert(ctx, user); err != nil {
		s.logger.Error("failed to upsert hydrated local user", "userID", user.ID(), "error", err)
		return nil, ErrInternalProcessing
	}

	s.logger.Info("[AUTH][DESKTOP] Usuario local hidratado correctamente", "email", user.Email(), "userID", user.ID())

	return user, nil
}

func normalizeRemoteProfile(remoteProfile *RemoteUserProfile, localUser *domain.User, email string) {
	if remoteProfile == nil {
		return
	}

	profileFirstName := strings.TrimSpace(remoteProfile.FirstName)
	profileLastName := strings.TrimSpace(remoteProfile.LastName)
	profileBirthDate := remoteProfile.BirthDate
	profileAvatarLocalPath := strings.TrimSpace(remoteProfile.AvatarLocalPath)

	if localUser != nil {
		localProfile := localUser.Profile()
		if profileFirstName == "" {
			profileFirstName = localProfile.FirstName()
		}
		if profileLastName == "" {
			profileLastName = localProfile.LastName()
		}
		if profileBirthDate.IsZero() {
			profileBirthDate = localProfile.BirthDate()
		}
		if profileAvatarLocalPath == "" {
			profileAvatarLocalPath = localUser.AvatarLocalPath()
		}
	}

	if profileFirstName == "" {
		profileFirstName = "Taskify"
	}
	if profileLastName == "" {
		profileLastName = fallbackRemoteLastName(email)
	}
	if profileBirthDate.IsZero() {
		profileBirthDate = time.Now().AddDate(-25, 0, 0).UTC()
	}

	remoteProfile.FirstName = profileFirstName
	remoteProfile.LastName = profileLastName
	remoteProfile.BirthDate = profileBirthDate
	remoteProfile.AvatarLocalPath = profileAvatarLocalPath
}

func fallbackRemoteLastName(email string) string {
	localPart := strings.TrimSpace(strings.Split(strings.TrimSpace(email), "@")[0])
	if len(localPart) >= 2 {
		return localPart
	}
	return "User"
}

func (s *userService) issueLocalSession(ctx context.Context, user *domain.User) (string, string, error) {
	tokenPair, err := s.tokenGen.GenerateTokenPair(tokenSubjectFromUser(user))
	if err != nil {
		s.logger.Error("failed to generate token pair", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	refreshSession, err := s.buildRefreshSession(user.ID(), tokenPair)
	if err != nil {
		s.logger.Error("failed to build refresh session", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	if err := s.sessionRepo.Save(ctx, refreshSession); err != nil {
		s.logger.Error("failed to save refresh session", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	return tokenPair.AccessToken, tokenPair.RefreshToken, nil
}

func (s *userService) RefreshSession(ctx context.Context, refreshToken string) (string, string, error) {
	if strings.TrimSpace(refreshToken) == "" {
		s.logger.Warn("refresh session rejected because token is empty")
		return "", "", ErrInvalidRefreshToken
	}

	refreshTokenHash := hashRefreshToken(refreshToken)
	currentSession, err := s.sessionRepo.GetByTokenHash(ctx, refreshTokenHash)
	if errors.Is(err, ports.ErrSessionNotFound) {
		s.logger.Warn("refresh session rejected because token was not found")
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		s.logger.Error("failed to retrieve refresh session", "error", err)
		return "", "", ErrInternalProcessing
	}
	if currentSession == nil {
		s.logger.Warn("refresh session rejected because token was not found")
		return "", "", ErrInvalidRefreshToken
	}
	if currentSession.IsRevoked() {
		s.logger.Warn("refresh session rejected because token is revoked", "sessionID", currentSession.ID())
		return "", "", ErrSessionRevoked
	}
	if currentSession.IsExpired(time.Now()) {
		s.logger.Warn("refresh session rejected because token is expired", "sessionID", currentSession.ID())
		return "", "", ErrRefreshSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, currentSession.UserID())
	if errors.Is(err, ports.ErrUserNotFound) || user == nil {
		s.logger.Warn("refresh session rejected because user was not found", "userID", currentSession.UserID())
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		s.logger.Error("failed to retrieve user during refresh", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	tokenPair, err := s.tokenGen.GenerateTokenPair(tokenSubjectFromUser(user))
	if err != nil {
		s.logger.Error("failed to generate refreshed token pair", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	newRefreshSession, err := s.buildRefreshSession(currentSession.UserID(), tokenPair)
	if err != nil {
		s.logger.Error("failed to build refreshed session", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	if err := s.sessionRepo.Rotate(ctx, currentSession.ID(), newRefreshSession); err != nil {
		s.logger.Error("failed to rotate refresh session", "sessionID", currentSession.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	return tokenPair.AccessToken, tokenPair.RefreshToken, nil
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if errors.Is(err, ports.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		s.logger.Error("failed to retrieve user profile", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}
	return user, nil
}

func (s *userService) UpdateAvatarLocalPath(ctx context.Context, userID, avatarLocalPath string) (*domain.User, error) {
	avatarLocalPath = strings.TrimSpace(avatarLocalPath)
	if avatarLocalPath == "" {
		return nil, ErrInvalidAvatarPath
	}

	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		if errors.Is(err, ports.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("failed to retrieve user before avatar update", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	if err := s.userRepo.UpdateAvatarLocalPath(ctx, userID, avatarLocalPath); err != nil {
		s.logger.Error("failed to update user avatar local path", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	return s.GetProfile(ctx, userID)
}

func (s *userService) UpdateProfileName(ctx context.Context, userID, name string) (*domain.User, error) {
	currentUser, err := s.userRepo.GetByID(ctx, userID)
	if errors.Is(err, ports.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		s.logger.Error("failed to retrieve user before profile update", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	firstName, lastName := splitDisplayName(name, currentUser.Profile().LastName())
	if _, err := domain.NewUserProfile(firstName, lastName, currentUser.Profile().BirthDate()); err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateProfileName(ctx, userID, firstName, lastName); err != nil {
		if errors.Is(err, ports.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("failed to update user profile name", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	return s.GetProfile(ctx, userID)
}

func (s *userService) buildRefreshSession(userID string, tokenPair ports.TokenPair) (*domain.RefreshToken, error) {
	sessionID := s.idGen.Generate()
	refreshTokenHash := hashRefreshToken(tokenPair.RefreshToken)
	return domain.NewRefreshToken(sessionID, userID, refreshTokenHash, tokenPair.RefreshTokenExpiresAt, false)
}

func splitDisplayName(name, fallbackLastName string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", strings.TrimSpace(fallbackLastName)
	}
	if len(parts) == 1 {
		return parts[0], strings.TrimSpace(fallbackLastName)
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func hashRefreshToken(refreshToken string) string {
	refreshTokenHash := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(refreshTokenHash[:])
}

func tokenSubjectFromUser(user *domain.User) ports.TokenSubject {
	profile := user.Profile()
	return ports.TokenSubject{
		UserID:    user.ID(),
		Email:     user.Email(),
		FirstName: profile.FirstName(),
		LastName:  profile.LastName(),
	}
}
