package user

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
	userrepo "github.com/dionisvl/avi/api-go/internal/repository/user"
)

type AvatarReader interface {
	GetUserAvatarURL(ctx context.Context, userID uuid.UUID) string
	GetUserAvatarObjectKey(ctx context.Context, userID uuid.UUID) (string, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

type UpdateMeInput struct {
	Name        *string
	Preferences *model.UserPreferences
}

type Service interface {
	GetMe(ctx context.Context, userID uuid.UUID) (*model.User, error)
	UpdateMe(ctx context.Context, userID uuid.UUID, in UpdateMeInput) (*model.User, error)
	DeleteMe(ctx context.Context, userID uuid.UUID, password string) error
}

type service struct {
	repo    userrepo.Repository
	avatars AvatarReader
	db      dbtx.TxBeginner
	logger  *slog.Logger
}

func New(repo userrepo.Repository, avatars AvatarReader, db dbtx.TxBeginner, logger *slog.Logger) Service {
	return &service{repo: repo, avatars: avatars, db: db, logger: logger}
}

func (s *service) GetMe(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	// Reuse user already loaded by auth middleware to avoid a redundant DB hit.
	if cached, ok := ctx.Value(apimiddleware.ContextKeyUser).(*model.User); ok && cached != nil && cached.ID == userID {
		cached.AvatarURL = s.avatars.GetUserAvatarURL(ctx, userID)
		return cached, nil
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, apierr.New(apierr.ErrUserNotFound, "User not found")
	}
	user.AvatarURL = s.avatars.GetUserAvatarURL(ctx, userID)
	return user, nil
}

func (s *service) UpdateMe(ctx context.Context, userID uuid.UUID, in UpdateMeInput) (*model.User, error) {
	if err := s.repo.Update(ctx, userID, userrepo.UpdateInput{
		Name:        in.Name,
		Preferences: in.Preferences,
	}); err != nil {
		s.logger.Error("update user", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to update profile")
	}
	return s.GetMe(ctx, userID)
}

func (s *service) DeleteMe(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return apierr.New(apierr.ErrUserNotFound, "User not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return apierr.New(apierr.ErrInvalidCredentials, "Invalid password")
	}

	avatarObjectKey, err := s.avatars.GetUserAvatarObjectKey(ctx, userID)
	if err != nil {
		s.logger.Error("get user avatar object", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to delete account")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("begin tx", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to delete account")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := userrepo.New(tx)
	if err := txRepo.PrepareAccountDeletion(ctx, userID); err != nil {
		s.logger.Error("prepare account deletion", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to delete account")
	}

	if err := txRepo.Delete(ctx, userID); err != nil {
		s.logger.Error("delete user", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to delete account")
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("commit delete account", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to delete account")
	}

	if avatarObjectKey != "" {
		if err := s.avatars.DeleteObject(ctx, avatarObjectKey); err != nil {
			s.logger.Error("delete user avatar object", slog.String("error", err.Error()))
		}
	}
	return nil
}
