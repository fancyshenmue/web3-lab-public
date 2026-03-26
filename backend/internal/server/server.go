package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/config"
	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/internal/handlers"
	"github.com/web3-lab/backend/internal/services"
	"github.com/web3-lab/backend/pkg/logs"
)

// Server is the HTTP application server.
type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	pool       *pgxpool.Pool
	redis      *redis.Client
	cfg        *config.Config

	healthHandler   *handlers.HealthHandler
	authHandler     *handlers.AuthHandler
	accountHandler  *handlers.AccountHandler
	oauth2Handler   *handlers.OAuth2Handler
	authzHandler    *handlers.AuthzHandler
	clientHandler   *handlers.ClientHandler
	siweHandler        *handlers.SIWEHandler
	templateHandler    *handlers.MessageTemplateHandler
	smartWalletHandler *handlers.SmartWalletHandler
}

// New creates and wires together all dependencies.
func New(cfg *config.Config) (*Server, error) {
	ctx := context.Background()
	gin.SetMode(cfg.Server.GinMode)

	// --- Postgres ---
	pool, err := pgxpool.New(ctx, cfg.Database.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	logs.Logger.Info("PostgreSQL connected")

	// --- Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Database,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	logs.Logger.Info("Redis connected")

	// --- SpiceDB (optional) ---
	var authzSvc *services.AuthzService
	if cfg.SpiceDB.Endpoint != "" && cfg.SpiceDB.Token != "" {
		authzSvc, err = services.NewAuthzService(cfg.SpiceDB.Endpoint, cfg.SpiceDB.Token, cfg.SpiceDB.Insecure)
		if err != nil {
			logs.Logger.Warn("SpiceDB unavailable, authorization features disabled", zap.Error(err))
		} else {
			logs.Logger.Info("SpiceDB connected")
		}
	}

	// --- Repository ---
	repo := database.NewPostgresRepository(pool)

	// --- Services ---
	authService := services.NewAuthService()
	nonceService := services.NewNonceService(redisClient, cfg.Redis.NonceKeyPrefix, cfg.Redis.NonceTTL)
	kratosService := services.NewKratosAdminService(cfg.Auth.KratosAdminURL, cfg.Auth.KratosPublicURL)
	hydraService := services.NewHydraClientService(cfg.Auth.HydraAdminURL, cfg.Auth.HydraPublicURL)
	accountService := services.NewAccountService(repo)
	walletAuthService := services.NewWalletAuthService(accountService, nonceService, authService, kratosService)
	appClientService := services.NewAppClientService(repo, repo, hydraService, redisClient, cfg.Redis.KeyPrefix)

	// --- Handlers ---
	healthHandler := handlers.NewHealthHandler(pool, redisClient, cfg.Server.Environment)
	authHandler := handlers.NewAuthHandler(walletAuthService, nonceService,
		cfg.Auth.WalletAuthDomain, cfg.Auth.WalletAuthVersion, cfg.Auth.WalletAuthChainID)
	accountHandler := handlers.NewAccountHandler(accountService)
	oauth2Handler := handlers.NewOAuth2Handler(hydraService, kratosService, appClientService, accountService)
	clientHandler := handlers.NewClientHandler(appClientService)

	var authzHandler *handlers.AuthzHandler
	if authzSvc != nil {
		authzHandler = handlers.NewAuthzHandler(authzSvc)
	}

	// --- Message Templates + SIWE ---
	templateService := services.NewMessageTemplateService(repo, redisClient, cfg.Redis.KeyPrefix)
	siweService := services.NewSIWEService(
		templateService, nonceService, authService, accountService, kratosService, hydraService,
		cfg.Auth.SIWEDomain, cfg.Auth.SIWEURI, cfg.Auth.SIWEStatement, cfg.Auth.SIWEVersion, cfg.Auth.SIWEChainID,
	)
	siweHandler := handlers.NewSIWEHandler(siweService)
	templateHandler := handlers.NewMessageTemplateHandler(templateService)

	smartWalletService, _ := services.NewSmartWalletService(cfg.Web3)
	bundlerService, _ := services.NewBundlerService(cfg.Web3)
	
	smartWalletHandler := handlers.NewSmartWalletHandler(smartWalletService, bundlerService)

	// --- Router ---
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.Use(zapMiddleware())
	router.Use(recoveryMiddleware())
	router.Use(corsMiddleware(cfg.Auth.CORSAllowedOrigins, appClientService))

	srv := &Server{
		router:          router,
		pool:            pool,
		redis:           redisClient,
		cfg:             cfg,
		healthHandler:   healthHandler,
		authHandler:     authHandler,
		accountHandler:  accountHandler,
		oauth2Handler:   oauth2Handler,
		authzHandler:    authzHandler,
		clientHandler:      clientHandler,
		siweHandler:        siweHandler,
		templateHandler:    templateHandler,
		smartWalletHandler: smartWalletHandler,
		httpServer: &http.Server{
			Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:        router,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}

	srv.setupRoutes()
	return srv, nil
}

// Start begins listening.
func (s *Server) Start() error {
	logs.Logger.Info("Server starting",
		zap.String("addr", s.httpServer.Addr),
		zap.String("env", s.cfg.Server.Environment),
	)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down all connections.
func (s *Server) Shutdown(ctx context.Context) error {
	logs.Logger.Info("Shutting down...")
	s.pool.Close()
	_ = s.redis.Close()
	return s.httpServer.Shutdown(ctx)
}
