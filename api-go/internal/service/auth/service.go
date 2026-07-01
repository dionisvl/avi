package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/dionisvl/avi/api-go/internal/config"
	emailpkg "github.com/dionisvl/avi/api-go/internal/email"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	sessionrepo "github.com/dionisvl/avi/api-go/internal/repository/session"
)

type EmailSender interface {
	SendVerificationCode(ctx context.Context, locale, to, code string) error
	SendPasswordResetCode(ctx context.Context, locale, to, code string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateVerifyCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
	SetResetCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error
	ClearResetCode(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	IncrementTokenVersion(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service interface {
	Register(ctx context.Context, req RegisterInput) (*RegisterOutput, error)
	ResendVerification(ctx context.Context, email, locale string) error
	Login(ctx context.Context, req LoginInput) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	VerifyEmail(ctx context.Context, email, code string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ConfirmPasswordReset(ctx context.Context, email, code string) error
	SetNewPassword(ctx context.Context, email, code, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	ValidateAccessToken(ctx context.Context, tokenString string) (*model.User, *Claims, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

const emailVerifyCodeTTL = 24 * time.Hour

type service struct {
	userRepo    UserRepository
	sessionRepo sessionrepo.Repository
	tokenSvc    *tokenService
	emailSvc    EmailSender
	logger      *slog.Logger
}

type RegisterInput struct {
	Email         string
	Password      string
	InitialRoles  []string
	EmailVerified bool
	Locale        string
}

type RegisterOutput struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

type LoginInput struct {
	Email    string
	Password string
}

func New(userRepo UserRepository, sessionRepo sessionrepo.Repository, emailSvc EmailSender, cfg *config.Config, logger *slog.Logger) Service {
	tokenSvc := newTokenService(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	SetResendCooldown(cfg.Auth.ResendCooldown)

	return &service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenSvc:    tokenSvc,
		emailSvc:    emailSvc,
		logger:      logger,
	}
}

func (s *service) Register(ctx context.Context, req RegisterInput) (*RegisterOutput, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		if existing.IsEmailVerified {
			return nil, apierr.New(apierr.ErrUserAlreadyExists, "An account with this email already exists")
		}
		return nil, apierr.NewWithCode(apierr.ErrUserExistsUnverified, apierr.CodeUserExistsUnverified, "An account with this email exists but the email is not yet verified")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to process password")
	}

	// Generate verification code
	verifyCode := generateCode()

	userID, err := uuid.NewV7()
	if err != nil {
		return nil, apierr.New(apierr.ErrInternal, "Failed to generate user ID")
	}

	user := model.NewUser(model.NewUserInput{
		ID:               userID,
		Email:            req.Email,
		PasswordHash:     string(hash),
		Locale:           emailpkg.PickLocale(req.Locale),
		VerifyCode:       verifyCode,
		EmailVerified:    req.EmailVerified,
		VerifyCodeExpiry: time.Now().Add(emailVerifyCodeTTL),
	})

	// Override roles if caller provided them (admin-initiated registration).
	if len(req.InitialRoles) > 0 {
		user.Roles = model.UserRolesFromStrings(req.InitialRoles)
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create user", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to create account")
	}

	if !req.EmailVerified {
		// Send verification email (5s timeout)
		emailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.emailSvc.SendVerificationCode(emailCtx, user.Locale, user.Email, verifyCode); err != nil {
			s.logger.Error("failed to send verification email", slog.String("error", err.Error()))
			return nil, apierr.New(apierr.ErrInternal, "Failed to send verification email: "+err.Error())
		}
	}

	return &RegisterOutput{
		ID:    user.ID,
		Email: user.Email,
		Roles: user.Roles.ToStrings(),
	}, nil
}

func (s *service) Login(ctx context.Context, req LoginInput) (*TokenPair, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, apierr.New(apierr.ErrInvalidCredentials, "Invalid email or password")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apierr.New(apierr.ErrInvalidCredentials, "Invalid email or password")
	}

	// Check if email is verified
	if !user.IsEmailVerified {
		return nil, apierr.New(apierr.ErrEmailNotVerified, "Please verify your email before logging in")
	}

	// Generate token pair
	pair, err := s.tokenSvc.GenerateTokenPair(user.ID, user.Email, user.Roles.ToStrings(), user.TokenVersion)
	if err != nil {
		return nil, apierr.New(apierr.ErrInternal, "Failed to generate token")
	}
	if err := s.sessionRepo.Create(ctx, pair.RefreshJTI, user.ID, pair.RefreshExp); err != nil {
		s.logger.Error("failed to create refresh session", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to create session")
	}
	return pair.TokenPair, nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.tokenSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, apierr.New(apierr.ErrInvalidToken, "Invalid or expired refresh token")
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, apierr.New(apierr.ErrInvalidToken, "Invalid or expired refresh token")
	}

	sess, err := s.sessionRepo.GetByJTI(ctx, jti)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierr.New(apierr.ErrInvalidToken, "Invalid or expired refresh token")
	}
	if err != nil {
		s.logger.Error("failed to get refresh session", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to validate session")
	}

	if sess.IsReuseAttempt() {
		s.logger.Warn("refresh token reuse detected, revoking all sessions", slog.String("user_id", sess.UserID.String()))
		_ = s.sessionRepo.RevokeAll(ctx, sess.UserID)
		return nil, apierr.New(apierr.ErrInvalidToken, "Invalid or expired refresh token")
	}
	if !sess.IsValid(time.Now()) {
		return nil, apierr.New(apierr.ErrInvalidToken, "Invalid or expired refresh token")
	}

	// Verify user still exists and token version is current
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, apierr.New(apierr.ErrInvalidToken, "User not found or token invalid")
	}
	if !user.HasValidTokenVersion(claims.TokenVersion) {
		return nil, apierr.New(apierr.ErrInvalidToken, "Token has been invalidated")
	}

	// Revoke old session (rotation)
	if err := s.sessionRepo.Revoke(ctx, jti); err != nil {
		s.logger.Error("failed to revoke old session", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to rotate session")
	}

	// Issue new token pair with new session
	pair, err := s.tokenSvc.GenerateTokenPair(user.ID, user.Email, user.Roles.ToStrings(), user.TokenVersion)
	if err != nil {
		return nil, apierr.New(apierr.ErrInternal, "Failed to generate token")
	}
	if err := s.sessionRepo.Create(ctx, pair.RefreshJTI, user.ID, pair.RefreshExp); err != nil {
		s.logger.Error("failed to create new refresh session", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to create session")
	}
	return pair.TokenPair, nil
}

func (s *service) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.sessionRepo.RevokeAll(ctx, userID); err != nil {
		s.logger.Error("failed to revoke refresh sessions on logout", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
	}
	if err := s.userRepo.IncrementTokenVersion(ctx, userID); err != nil {
		s.logger.Error("failed to invalidate tokens on logout", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "failed to log out")
	}
	return nil
}

func (s *service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't leak whether email exists
		return nil
	}

	resetCode := generateCode()
	expiry := time.Now().Add(15 * time.Minute)
	codeAttempts.reset(resetAttemptKey(user.Email))

	if err := s.userRepo.SetResetCode(ctx, user.ID, resetCode, expiry); err != nil {
		s.logger.Error("failed to set reset code", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to request password reset")
	}

	// Send reset email (with timeout); error is logged but not returned to avoid leaking email existence
	emailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.emailSvc.SendPasswordResetCode(emailCtx, user.Locale, email, resetCode); err != nil {
		s.logger.Error("failed to send reset email", slog.String("error", err.Error()))
	}

	return nil
}

func (s *service) ConfirmPasswordReset(ctx context.Context, email, code string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return apierr.New(apierr.ErrInvalidResetCode, "Invalid or expired reset code")
	}

	key := resetAttemptKey(email)
	if !codeAttempts.record(key) {
		return apierr.New(apierr.ErrInvalidResetCode, "Invalid or expired reset code")
	}

	if !user.IsResetCodeValid(code, time.Now()) {
		return apierr.New(apierr.ErrInvalidResetCode, "Invalid or expired reset code")
	}

	return nil
}

func (s *service) SetNewPassword(ctx context.Context, email, code, newPassword string) error {
	if err := s.ConfirmPasswordReset(ctx, email, code); err != nil {
		return err
	}

	user, _ := s.userRepo.GetByEmail(ctx, email)

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to process password")
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
		s.logger.Error("failed to update password", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to update password")
	}

	if err := s.userRepo.ClearResetCode(ctx, user.ID); err != nil {
		s.logger.Error("failed to clear reset code", slog.String("error", err.Error()))
	}
	codeAttempts.reset(resetAttemptKey(email))

	return nil
}

func (s *service) VerifyEmail(ctx context.Context, email, code string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return apierr.New(apierr.ErrInvalidVerificationCode, "Invalid verification code")
	}

	if user.IsEmailVerified {
		return apierr.New(apierr.ErrEmailAlreadyVerified, "Email is already verified")
	}

	key := verifyAttemptKey(email)
	if !codeAttempts.record(key) {
		return apierr.New(apierr.ErrInvalidVerificationCode, "Invalid or expired verification code")
	}

	if !user.IsEmailVerificationValid(code, time.Now()) {
		return apierr.New(apierr.ErrInvalidVerificationCode, "Invalid or expired verification code")
	}

	codeAttempts.reset(key)
	if err := s.userRepo.MarkEmailVerified(ctx, user.ID); err != nil {
		s.logger.Error("failed to mark email verified", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to verify email")
	}

	return nil
}

func (s *service) ResendVerification(ctx context.Context, email, locale string) error {
	key := resendCooldownKey(email)
	if !resendCooldowns.allow(key) {
		return apierr.New(apierr.ErrRateLimited, "Please wait before requesting another verification code")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user.IsEmailVerified {
		// Silently succeed — don't leak whether email exists or is already verified.
		return nil
	}

	newCode := generateCode()
	codeAttempts.reset(verifyAttemptKey(email))
	if err := s.userRepo.UpdateVerifyCode(ctx, user.ID, newCode, time.Now().Add(emailVerifyCodeTTL)); err != nil {
		s.logger.Error("failed to update verify code", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to resend verification code")
	}

	emailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pickedLocale := emailpkg.PickLocale(locale)
	if err := s.emailSvc.SendVerificationCode(emailCtx, pickedLocale, email, newCode); err != nil {
		s.logger.Error("failed to send verification email", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to send verification email")
	}

	return nil
}

func (s *service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		s.logger.Error("failed to delete user", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to rollback account creation")
	}
	return nil
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apierr.New(apierr.ErrUserNotFound, "User not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return apierr.NewWithCode(apierr.ErrInvalidCredentials, apierr.CodeWrongCurrentPassword, "Current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to process password")
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		s.logger.Error("failed to update password", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to update password")
	}

	return nil
}

func (s *service) ValidateAccessToken(ctx context.Context, tokenString string) (*model.User, *Claims, error) {
	claims, err := s.tokenSvc.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, nil, apierr.ErrInvalidToken
	}
	if !user.HasValidTokenVersion(claims.TokenVersion) {
		return nil, nil, apierr.ErrInvalidToken
	}

	return user, claims, nil
}

func generateCode() string {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			n = big.NewInt(time.Now().UnixNano() % 10)
		}
		code[i] = digits[n.Int64()]
	}
	return string(code)
}
