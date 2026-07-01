package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/dionisvl/avi/api-go/internal/api/auth"
	categoriesapi "github.com/dionisvl/avi/api-go/internal/api/categories"
	chatapi "github.com/dionisvl/avi/api-go/internal/api/chat"
	cityapi "github.com/dionisvl/avi/api-go/internal/api/city"
	contactapi "github.com/dionisvl/avi/api-go/internal/api/contact"
	favoriteapi "github.com/dionisvl/avi/api-go/internal/api/favorite"
	"github.com/dionisvl/avi/api-go/internal/api/health"
	itemapi "github.com/dionisvl/avi/api-go/internal/api/item"
	paymentapi "github.com/dionisvl/avi/api-go/internal/api/payment"
	uploadapi "github.com/dionisvl/avi/api-go/internal/api/upload"
	userapi "github.com/dionisvl/avi/api-go/internal/api/user"
	"github.com/dionisvl/avi/api-go/internal/config"
	"github.com/dionisvl/avi/api-go/internal/email"
	"github.com/dionisvl/avi/api-go/internal/migrations"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/payment/provider"
	"github.com/dionisvl/avi/api-go/internal/payment/yookassa"
	categoryquery "github.com/dionisvl/avi/api-go/internal/query/category"
	chatquery "github.com/dionisvl/avi/api-go/internal/query/chatview"
	favoritequery "github.com/dionisvl/avi/api-go/internal/query/favoriteview"
	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
	categoryrepo "github.com/dionisvl/avi/api-go/internal/repository/category"
	chatrepo "github.com/dionisvl/avi/api-go/internal/repository/chat"
	cityrepo "github.com/dionisvl/avi/api-go/internal/repository/city"
	favrepo "github.com/dionisvl/avi/api-go/internal/repository/favorite"
	itemrepo "github.com/dionisvl/avi/api-go/internal/repository/item"
	mediarepo "github.com/dionisvl/avi/api-go/internal/repository/media"
	paymentrepo "github.com/dionisvl/avi/api-go/internal/repository/payment"
	sessionrepo "github.com/dionisvl/avi/api-go/internal/repository/session"
	userrepo "github.com/dionisvl/avi/api-go/internal/repository/user"
	authservice "github.com/dionisvl/avi/api-go/internal/service/auth"
	chatservice "github.com/dionisvl/avi/api-go/internal/service/chat"
	contactservice "github.com/dionisvl/avi/api-go/internal/service/contact"
	favoriteservice "github.com/dionisvl/avi/api-go/internal/service/favorite"
	itemservice "github.com/dionisvl/avi/api-go/internal/service/item"
	mediaservice "github.com/dionisvl/avi/api-go/internal/service/media"
	paymentservice "github.com/dionisvl/avi/api-go/internal/service/payment"
	userservice "github.com/dionisvl/avi/api-go/internal/service/user"
	"github.com/dionisvl/avi/api-go/internal/storage"
)

type diContainer struct {
	cfg    *config.Config
	logger *slog.Logger

	db          *pgxpool.Pool
	storage     objectStorage
	emailSender email.Sender

	userRepo     userrepo.Repository
	mediaRepo    mediarepo.Repository
	sessionRepo  sessionrepo.Repository
	itemRepo     itemrepo.Repository
	cityRepo     cityrepo.Repository
	categoryRepo categoryrepo.Repository
	favoriteRepo favrepo.Repository
	chatRepo     chatrepo.Repository
	paymentRepo  paymentrepo.Repository

	authSvc      authservice.Service
	mediaSvc     mediaservice.Service
	itemSvc      itemservice.Service
	itemQuery    itemquery.Service
	categoryView categoryquery.Service
	userSvc      userservice.Service
	favoriteSvc  favoriteservice.Service
	favoriteView favoritequery.Service
	contactSvc   contactservice.Service
	chatSvc      chatservice.Service
	chatView     chatquery.Service
	paymentSvc   paymentservice.Service

	healthHandler     *health.Handler
	authHandler       *auth.Handler
	userHandler       *userapi.Handler
	uploadHandler     *uploadapi.Handler
	itemHandler       *itemapi.Handler
	cityHandler       *cityapi.Handler
	categoriesHandler *categoriesapi.Handler
	favoriteHandler   *favoriteapi.Handler
	contactHandler    *contactapi.Handler
	chatHandler       *chatapi.Handler
	chatHub           *chatapi.Hub
	paymentHandler    *paymentapi.Handler
}

type objectStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) (string, error)
	Delete(ctx context.Context, key string) error
}

func newDIContainer(cfg *config.Config, logger *slog.Logger) *diContainer {
	return &diContainer{
		cfg:    cfg,
		logger: logger,
	}
}

func (c *diContainer) DB() *pgxpool.Pool {
	if c.db != nil {
		return c.db
	}

	poolCfg, err := pgxpool.ParseConfig(c.cfg.DB.DSN)
	if err != nil {
		c.logger.Error("failed to parse db dsn", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = c.cfg.DB.MaxConns
	poolCfg.MinConns = c.cfg.DB.MinConns
	poolCfg.MaxConnLifetime = c.cfg.DB.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		c.logger.Error("failed to create db pool", "error", err)
		os.Exit(1)
	}

	if err := pool.Ping(context.Background()); err != nil {
		c.logger.Error("failed to ping db", "error", err)
		os.Exit(1)
	}

	c.logger.Info("database connected successfully")
	c.db = pool

	return c.db
}

func (c *diContainer) Migrate() {
	db := stdlib.OpenDB(*c.DB().Config().ConnConfig)
	defer func() {
		if err := db.Close(); err != nil {
			c.logger.Error("failed to close db", "error", err)
		}
	}()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		c.logger.Error("failed to set dialect", "error", err)
		os.Exit(1)
	}

	if err := goose.UpContext(context.Background(), db, "."); err != nil {
		c.logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	c.logger.Info("migrations applied successfully")
}

func (c *diContainer) UserRepo() userrepo.Repository {
	if c.userRepo != nil {
		return c.userRepo
	}
	c.userRepo = userrepo.New(c.DB())
	return c.userRepo
}

func (c *diContainer) SessionRepo() sessionrepo.Repository {
	if c.sessionRepo != nil {
		return c.sessionRepo
	}
	c.sessionRepo = sessionrepo.New(c.DB())
	return c.sessionRepo
}

func (c *diContainer) AuthSvc() authservice.Service {
	if c.authSvc != nil {
		return c.authSvc
	}
	c.authSvc = authservice.New(c.UserRepo(), c.SessionRepo(), c.EmailSender(), c.cfg, c.logger)
	return c.authSvc
}

func (c *diContainer) EmailSender() email.Sender {
	if c.emailSender != nil {
		return c.emailSender
	}
	c.emailSender = email.New(
		c.cfg.SMTP.Host,
		c.cfg.SMTP.Port,
		c.cfg.SMTP.From,
		c.cfg.SMTP.User,
		c.cfg.SMTP.Password,
		c.cfg.SMTP.FrontendDomain,
		c.logger,
	)
	return c.emailSender
}

func (c *diContainer) HealthHandler() *health.Handler {
	if c.healthHandler != nil {
		return c.healthHandler
	}
	c.healthHandler = health.NewHandler(config.Version)
	return c.healthHandler
}

func (c *diContainer) AuthHandler() *auth.Handler {
	if c.authHandler != nil {
		return c.authHandler
	}
	c.authHandler = auth.NewHandler(c.AuthSvc(), c.cfg.Auth, c.cfg.App, c.logger)
	return c.authHandler
}

func (c *diContainer) UserSvc() userservice.Service {
	if c.userSvc != nil {
		return c.userSvc
	}
	c.userSvc = userservice.New(c.UserRepo(), c.MediaSvc(), c.DB(), c.logger)
	return c.userSvc
}

func (c *diContainer) UserHandler() *userapi.Handler {
	if c.userHandler != nil {
		return c.userHandler
	}
	c.userHandler = userapi.NewHandler(c.AuthSvc(), c.UserSvc(), c.logger)
	return c.userHandler
}

func (c *diContainer) MediaRepo() mediarepo.Repository {
	if c.mediaRepo != nil {
		return c.mediaRepo
	}
	c.mediaRepo = mediarepo.New(c.DB())
	return c.mediaRepo
}

func (c *diContainer) ObjectStorage() objectStorage {
	if c.storage != nil {
		return c.storage
	}

	if storage.IsLocalEndpoint(c.cfg.S3.Endpoint) {
		local, err := storage.NewLocalStorage(storage.LocalRootFromEndpoint(c.cfg.S3.Endpoint))
		if err != nil {
			c.logger.Error("failed to initialize local object storage", "error", err)
			os.Exit(1)
		}
		c.storage = local
		return c.storage
	}

	c.storage = storage.NewS3Client(
		c.cfg.S3.Endpoint,
		c.cfg.S3.Region,
		c.cfg.S3.KeyID,
		c.cfg.S3.KeySecret,
		c.cfg.S3.Bucket,
	)
	return c.storage
}

func (c *diContainer) MediaSvc() mediaservice.Service {
	if c.mediaSvc != nil {
		return c.mediaSvc
	}
	c.mediaSvc = mediaservice.New(
		c.MediaRepo(),
		c.ObjectStorage(),
		c.cfg.S3.Bucket,
		c.cfg.S3.PublicBaseURL,
		c.logger,
	)
	return c.mediaSvc
}

func (c *diContainer) UploadHandler() *uploadapi.Handler {
	if c.uploadHandler != nil {
		return c.uploadHandler
	}
	c.uploadHandler = uploadapi.NewHandler(c.MediaSvc(), c.ItemSvc(), c.logger)
	return c.uploadHandler
}

func (c *diContainer) ItemRepo() itemrepo.Repository {
	if c.itemRepo != nil {
		return c.itemRepo
	}
	c.itemRepo = itemrepo.New(c.DB())
	return c.itemRepo
}

func (c *diContainer) CategoryRepo() categoryrepo.Repository {
	if c.categoryRepo != nil {
		return c.categoryRepo
	}
	c.categoryRepo = categoryrepo.New(c.DB())
	return c.categoryRepo
}

func (c *diContainer) CategoryQuery() categoryquery.Service {
	if c.categoryView != nil {
		return c.categoryView
	}
	c.categoryView = categoryquery.New(c.CategoryRepo())
	return c.categoryView
}

func (c *diContainer) CategoriesHandler() *categoriesapi.Handler {
	if c.categoriesHandler != nil {
		return c.categoriesHandler
	}
	c.categoriesHandler = categoriesapi.NewHandler(c.CategoryQuery(), c.logger)
	return c.categoriesHandler
}

func (c *diContainer) CityRepo() cityrepo.Repository {
	if c.cityRepo != nil {
		return c.cityRepo
	}
	c.cityRepo = cityrepo.New(c.DB())
	return c.cityRepo
}

func (c *diContainer) CityHandler() *cityapi.Handler {
	if c.cityHandler != nil {
		return c.cityHandler
	}
	c.cityHandler = cityapi.NewHandler(c.CityRepo(), c.logger)
	return c.cityHandler
}

func (c *diContainer) ItemSvc() itemservice.Service {
	if c.itemSvc != nil {
		return c.itemSvc
	}
	c.itemSvc = itemservice.New(c.ItemRepo(), c.CategoryRepo(), c.MediaRepo(), c.DB(), c.logger)
	return c.itemSvc
}

func (c *diContainer) FavoriteRepo() favrepo.Repository {
	if c.favoriteRepo != nil {
		return c.favoriteRepo
	}
	c.favoriteRepo = favrepo.New(c.DB())
	return c.favoriteRepo
}

func (c *diContainer) FavoriteSvc() favoriteservice.Service {
	if c.favoriteSvc != nil {
		return c.favoriteSvc
	}
	c.favoriteSvc = favoriteservice.New(c.FavoriteRepo(), c.logger)
	return c.favoriteSvc
}

func (c *diContainer) ItemQuery() itemquery.Service {
	if c.itemQuery != nil {
		return c.itemQuery
	}
	c.itemQuery = itemquery.New(c.ItemRepo(), c.FavoriteSvc(), c.cfg.S3.PublicBaseURL)
	return c.itemQuery
}

func (c *diContainer) ItemHandler() *itemapi.Handler {
	if c.itemHandler != nil {
		return c.itemHandler
	}
	c.itemHandler = itemapi.NewHandler(c.ItemSvc(), c.ItemQuery(), c.CityRepo(), c.logger)
	return c.itemHandler
}

func (c *diContainer) FavoriteView() favoritequery.Service {
	if c.favoriteView != nil {
		return c.favoriteView
	}
	c.favoriteView = favoritequery.New(c.FavoriteRepo(), c.cfg.S3.PublicBaseURL)
	return c.favoriteView
}

func (c *diContainer) FavoriteHandler() *favoriteapi.Handler {
	if c.favoriteHandler != nil {
		return c.favoriteHandler
	}
	c.favoriteHandler = favoriteapi.NewHandler(c.FavoriteSvc(), c.FavoriteView(), c.logger)
	return c.favoriteHandler
}

func (c *diContainer) ContactSvc() contactservice.Service {
	if c.contactSvc != nil {
		return c.contactSvc
	}
	c.contactSvc = contactservice.New(c.EmailSender(), c.cfg.SMTP.ContactTo, c.logger)
	return c.contactSvc
}

func (c *diContainer) ContactHandler() *contactapi.Handler {
	if c.contactHandler != nil {
		return c.contactHandler
	}
	c.contactHandler = contactapi.NewHandler(c.ContactSvc(), c.logger)
	return c.contactHandler
}

func (c *diContainer) ChatRepo() chatrepo.Repository {
	if c.chatRepo != nil {
		return c.chatRepo
	}
	c.chatRepo = chatrepo.New(c.DB())
	return c.chatRepo
}

func (c *diContainer) ChatHub() *chatapi.Hub {
	if c.chatHub != nil {
		return c.chatHub
	}
	c.chatHub = chatapi.NewHub(c.logger)
	return c.chatHub
}

func (c *diContainer) ChatSvc() chatservice.Service {
	if c.chatSvc != nil {
		return c.chatSvc
	}
	c.chatSvc = chatservice.New(c.ChatRepo(), c.UserRepo(), c.ChatHub(), c.cfg.S3.PublicBaseURL, c.logger)
	return c.chatSvc
}

func (c *diContainer) ChatView() chatquery.Service {
	if c.chatView != nil {
		return c.chatView
	}
	c.chatView = chatquery.New(c.ChatRepo(), c.cfg.S3.PublicBaseURL, c.UserRepo())
	return c.chatView
}

func (c *diContainer) ChatHandler() *chatapi.Handler {
	if c.chatHandler != nil {
		return c.chatHandler
	}
	c.chatHandler = chatapi.NewHandler(c.ChatSvc(), c.ChatView(), c.MediaSvc(), c.ChatHub(), c.cfg.S3.PublicBaseURL, c.cfg.App.TrustedOrigins, c.logger)
	return c.chatHandler
}

func (c *diContainer) PaymentRepo() paymentrepo.Repository {
	if c.paymentRepo != nil {
		return c.paymentRepo
	}
	c.paymentRepo = paymentrepo.New(c.DB())
	return c.paymentRepo
}

func (c *diContainer) PaymentProvider() provider.Provider {
	if !c.cfg.YooKassa.Enabled {
		c.logger.Warn("yookassa disabled, using noop provider")
		return &noopProvider{}
	}
	if c.cfg.YooKassa.ShopID == "" || c.cfg.YooKassa.SecretKey == "" {
		c.logger.Error("yookassa enabled but credentials are not configured")
		os.Exit(1)
	}

	return yookassa.NewAdapter(c.cfg.YooKassa.ShopID, c.cfg.YooKassa.SecretKey, yookassa.ReceiptConfig{
		VatCode:        c.cfg.Payments.ReceiptVatCode,
		PaymentSubject: c.cfg.Payments.ReceiptPaymentSubject,
		PaymentMode:    c.cfg.Payments.ReceiptPaymentMode,
	})
}

func (c *diContainer) PaymentSvc() paymentservice.Service {
	if c.paymentSvc != nil {
		return c.paymentSvc
	}
	c.paymentSvc = paymentservice.New(
		c.PaymentRepo(),
		c.PaymentProvider(),
		c.ItemSvc(),
		c.UserRepo(),
		c.cfg.Payments.PromoteListingAmountMinor,
		c.cfg.Payments.Currency,
		c.logger,
	)
	return c.paymentSvc
}

func (c *diContainer) PaymentHandler() *paymentapi.Handler {
	if c.paymentHandler != nil {
		return c.paymentHandler
	}
	c.paymentHandler = paymentapi.NewHandler(c.PaymentSvc(), c.cfg.Payments.ReturnURL, c.cfg.SMTP.FrontendDomain, c.logger)
	return c.paymentHandler
}

type noopProvider struct{}

func (n *noopProvider) CreatePayment(ctx context.Context, in provider.CreatePaymentInput) (*provider.CreatedPayment, error) {
	return &provider.CreatedPayment{
		ProviderPaymentID: "noop-" + in.LocalPaymentID.String(),
		Status:            "pending",
		ConfirmationURL:   "https://example.com/confirm",
		Metadata:          map[string]any{},
	}, nil
}

func (n *noopProvider) GetPayment(ctx context.Context, providerPaymentID string) (*provider.PaymentInfo, error) {
	return &provider.PaymentInfo{
		ProviderPaymentID: providerPaymentID,
		Status:            model.PaymentStatusPending,
		ProviderStatus:    "pending",
		Metadata:          map[string]any{},
	}, nil
}

func (n *noopProvider) ParseWebhookEvent(payload []byte) (*provider.WebhookEvent, error) {
	return nil, fmt.Errorf("noop provider does not support webhooks")
}
