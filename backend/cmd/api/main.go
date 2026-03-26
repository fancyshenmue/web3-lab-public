package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"

	"github.com/web3-lab/backend/internal/config"
	"github.com/web3-lab/backend/internal/server"
	"github.com/web3-lab/backend/pkg/logs"
)

var rootCmd = &cobra.Command{
	Use:   "api",
	Short: "Web3 Account API Server",
	Long:  "Backend API for the Web3-Lab identity and authorization platform.\nManages wallet authentication, account identities, and integrates with Ory and SpiceDB.",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := logs.Init(cfg.Server.Environment, "web3-account-api"); err != nil {
			return fmt.Errorf("init logger: %w", err)
		}
		defer logs.Sync()

		srv, err := server.New(cfg)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		// Graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			if err := srv.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server error: %v", err)
			}
		}()

		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn, _ := cmd.Flags().GetString("dsn")
		path, _ := cmd.Flags().GetString("path")
		if dsn == "" {
			dsn = os.Getenv("POSTGRES_DSN")
		}
		if dsn == "" {
			return fmt.Errorf("--dsn or POSTGRES_DSN is required")
		}

		m, err := migrate.New("file://"+path, dsn)
		if err != nil {
			return fmt.Errorf("init migrate: %w", err)
		}
		defer m.Close()

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate up: %w", err)
		}

		log.Println("Migrations applied successfully")
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn, _ := cmd.Flags().GetString("dsn")
		path, _ := cmd.Flags().GetString("path")
		steps, _ := cmd.Flags().GetInt("steps")
		if dsn == "" {
			dsn = os.Getenv("POSTGRES_DSN")
		}
		if dsn == "" {
			return fmt.Errorf("--dsn or POSTGRES_DSN is required")
		}

		m, err := migrate.New("file://"+path, dsn)
		if err != nil {
			return fmt.Errorf("init migrate: %w", err)
		}
		defer m.Close()

		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate down: %w", err)
		}

		log.Printf("Rolled back %d migration(s)", steps)
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn, _ := cmd.Flags().GetString("dsn")
		path, _ := cmd.Flags().GetString("path")
		if dsn == "" {
			dsn = os.Getenv("POSTGRES_DSN")
		}
		if dsn == "" {
			return fmt.Errorf("--dsn or POSTGRES_DSN is required")
		}

		m, err := migrate.New("file://"+path, dsn)
		if err != nil {
			return fmt.Errorf("init migrate: %w", err)
		}
		defer m.Close()

		version, dirty, err := m.Version()
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		log.Printf("Current version: %d (dirty: %v)", version, dirty)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)

	// Migration flags
	migrateCmd.PersistentFlags().String("dsn", "", "PostgreSQL connection string (or set POSTGRES_DSN)")
	migrateCmd.PersistentFlags().String("path", "./migrations/postgres", "Path to migration files")
	migrateDownCmd.Flags().Int("steps", 1, "Number of migrations to rollback")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
