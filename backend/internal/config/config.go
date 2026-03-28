package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	SpiceDB  SpiceDBConfig  `mapstructure:"spicedb"`
	Web3     Web3Config     `mapstructure:"web3"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
}

type ServerConfig struct {
	Port        int    `mapstructure:"port"`
	Host        string `mapstructure:"host"`
	GinMode     string `mapstructure:"gin_mode"`
	Environment string `mapstructure:"environment"`
}

type DatabaseConfig struct {
	PostgresDSN string `mapstructure:"postgres_dsn"`
}

type RedisConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Password        string        `mapstructure:"password"`
	Database        int           `mapstructure:"database"`
	PoolSize        int           `mapstructure:"pool_size"`
	KeyPrefix       string        `mapstructure:"key_prefix"`
	NonceKeyPrefix  string        `mapstructure:"nonce_key_prefix"`
	NonceTTLMinutes int           `mapstructure:"nonce_ttl_minutes"`
	NonceTTL        time.Duration // computed
}

type AuthConfig struct {
	KratosPublicURL string `mapstructure:"kratos_public_url"`
	KratosAdminURL  string `mapstructure:"kratos_admin_url"`
	HydraPublicURL  string `mapstructure:"hydra_public_url"`
	HydraAdminURL   string `mapstructure:"hydra_admin_url"`
	FrontendURL     string `mapstructure:"frontend_url"`

	WalletAuthDomain    string `mapstructure:"wallet_auth_domain"`
	WalletAuthVersion   string `mapstructure:"wallet_auth_version"`
	WalletAuthChainID   int    `mapstructure:"wallet_auth_chain_id"`
	WalletAuthStatement string `mapstructure:"wallet_auth_statement"`

	// SIWE config (fallback defaults when no message template is resolved)
	SIWEDomain    string `mapstructure:"siwe_domain"`
	SIWEURI       string `mapstructure:"siwe_uri"`
	SIWEStatement string `mapstructure:"siwe_statement"`
	SIWEChainID   int    `mapstructure:"siwe_chain_id"`
	SIWEVersion   string `mapstructure:"siwe_version"`

	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
}

type SpiceDBConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Token    string `mapstructure:"token"`
	Insecure bool   `mapstructure:"insecure"`
}

type Web3Config struct {
	GethRPCUrl     string `mapstructure:"geth_rpc_url"`
	EntryPointAddr string `mapstructure:"entry_point_address"`

	// Factories
	AccountFactoryAddr string `mapstructure:"account_factory_address"`
	ERC20FactoryAddr   string `mapstructure:"erc20_factory_address"`
	ERC721FactoryAddr  string `mapstructure:"erc721_factory_address"`
	ERC1155FactoryAddr string `mapstructure:"erc1155_factory_address"`

	// Paymaster
	PaymasterPriv string `mapstructure:"paymaster_private_key"`
	PaymasterAddr string `mapstructure:"paymaster_address"`

	// Genesis Mock Accounts (for local TEE generation)
	GenesisAddresses []string `mapstructure:"genesis_addresses"`
	GenesisKeys      []string `mapstructure:"genesis_keys"`
}

type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey  string `mapstructure:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	PublicBaseURL   string `mapstructure:"public_base_url"`
	InternalBaseURL string `mapstructure:"internal_base_url"`
	PresignedExpiry int    `mapstructure:"presigned_expiry_minutes"`
}

// Load reads config from file and env vars.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("./cmd/api")
	v.AddConfigPath("/etc/config/api")
	v.AutomaticEnv()

	// Bind key env vars
	_ = v.BindEnv("database.postgres_dsn", "POSTGRES_DSN")
	_ = v.BindEnv("spicedb.token", "SPICEDB_GRPC_PRESHARED_KEY")
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.gin_mode", "GIN_MODE")

	// Defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.gin_mode", "release")
	v.SetDefault("server.environment", "minikube")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.key_prefix", "web3")
	v.SetDefault("redis.nonce_key_prefix", "nonce")
	v.SetDefault("redis.nonce_ttl_minutes", 5)
	v.SetDefault("auth.wallet_auth_chain_id", 1)
	v.SetDefault("auth.siwe_domain", "app.web3-local-dev.com")
	v.SetDefault("auth.siwe_uri", "https://app.web3-local-dev.com")
	v.SetDefault("auth.siwe_statement", "Sign in to {service_name}")
	v.SetDefault("auth.siwe_chain_id", 1)
	v.SetDefault("auth.siwe_version", "1")
	v.SetDefault("spicedb.insecure", true)
	
	v.SetDefault("web3.geth_rpc_url", "http://geth-rpc:8545")
	v.SetDefault("web3.paymaster_private_key", "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		log.Println("No config file found, using env vars and defaults")
	} else {
		log.Printf("Config loaded from %s", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.Database.PostgresDSN == "" {
		dsn := os.Getenv("POSTGRES_DSN")
		if dsn != "" {
			cfg.Database.PostgresDSN = dsn
		} else {
			return nil, fmt.Errorf("POSTGRES_DSN is required")
		}
	}

	// Compute nonce TTL
	cfg.Redis.NonceTTL = time.Duration(cfg.Redis.NonceTTLMinutes) * time.Minute
	// Build full nonce key prefix
	cfg.Redis.NonceKeyPrefix = cfg.Redis.KeyPrefix + ":" + cfg.Server.Environment + ":" + cfg.Redis.NonceKeyPrefix

	return &cfg, nil
}
