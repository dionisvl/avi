package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	"github.com/dionisvl/avi/api-go/internal/config"
	"github.com/dionisvl/avi/api-go/internal/storage"
)

type App struct {
	cfg     *config.Config
	logger  *slog.Logger
	router  chi.Router
	di      *diContainer
	httpSrv *http.Server
}

func New() *App {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	a := &App{
		cfg:    cfg,
		logger: logger,
		di:     newDIContainer(cfg, logger),
	}

	a.initDeps()
	a.router = a.buildRouter()

	a.httpSrv = &http.Server{
		Addr:              cfg.App.Port,
		Handler:           a.router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return a
}

func (a *App) initDeps() {
	inits := []func(){
		func() { a.di.DB() },      // force-init and ping
		func() { a.di.Migrate() }, // run goose migrations
	}

	for _, fn := range inits {
		fn()
	}
}

func (a *App) buildRouter() chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimiddleware.Logger(a.logger))
	r.Use(apimiddleware.CORS(a.cfg))
	r.Use(apimiddleware.Locale)
	r.Use(middleware.Recoverer)

	// Health check (no auth required)
	r.Get("/health", a.di.HealthHandler().ServeHTTP)

	// Swagger UI + spec (see swagger_ui.go)
	a.mountSwagger(r)

	if storage.IsLocalEndpoint(a.cfg.S3.Endpoint) {
		root := storage.LocalRootFromEndpoint(a.cfg.S3.Endpoint)
		r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(root))))
	}

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", a.di.AuthHandler().Routes())
		r.Mount("/user", a.di.UserHandler().Routes())
		r.Mount("/upload", a.di.UploadHandler().Routes(a.di.AuthSvc()))
		r.Mount("/items", a.di.ItemHandler().Routes(a.di.AuthSvc()))
		r.Mount("/items/favorites", a.di.FavoriteHandler().Routes(a.di.AuthSvc()))
		r.Mount("/cities", a.di.CityHandler().Routes())
		r.Mount("/categories", a.di.CategoriesHandler().Routes())
		r.Mount("/chat", a.di.ChatHandler().Routes(a.di.AuthSvc()))
		r.Mount("/payments", a.di.PaymentHandler().Routes(a.di.AuthSvc()))
		r.Group(func(r chi.Router) {
			r.Use(apimiddleware.RateLimit(
				a.cfg.Auth.ContactRateLimitRPS,
				a.cfg.Auth.ContactRateLimitBurst,
				a.cfg.App.TrustedProxies...,
			))
			r.Mount("/contact-messages", a.di.ContactHandler().Routes())
		})
	})

	return r
}

func (a *App) Run() error {
	a.logger.Info("starting server",
		slog.String("addr", a.cfg.App.Port),
		slog.String("env", a.cfg.App.Env),
	)

	if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("server error", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.httpSrv.Shutdown(ctx)
}
