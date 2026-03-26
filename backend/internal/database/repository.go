package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// AccountRepository defines the interface for all account-related database operations.
// This mirrors the old Spanner repository contract so handlers remain unchanged.
type AccountRepository interface {
	// Account operations
	CreateAccount(ctx context.Context, account *Account) error
	GetAccountByID(ctx context.Context, accountID uuid.UUID) (*Account, error)
	GetAccountByKratosIdentityID(ctx context.Context, kratosIdentityID uuid.UUID) (*Account, error)
	UpdateAccount(ctx context.Context, account *Account) error
	UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status string) error
	UpdateLastLogin(ctx context.Context, accountID uuid.UUID, ts time.Time) error

	// Identity Provider operations
	GetIdentityProvider(ctx context.Context, providerID string) (*IdentityProvider, error)
	ListIdentityProviders(ctx context.Context, onlyEnabled bool) ([]*IdentityProvider, error)

	// Account Identity operations
	CreateAccountIdentity(ctx context.Context, identity *AccountIdentity) error
	GetAccountIdentity(ctx context.Context, identityID uuid.UUID) (*AccountIdentity, error)
	GetAccountIdentitiesByAccountID(ctx context.Context, accountID uuid.UUID) ([]*AccountIdentity, error)
	GetAccountIdentityByKratosID(ctx context.Context, kratosIdentityID uuid.UUID) (*AccountIdentity, error)
	GetAccountIdentityByProviderUserID(ctx context.Context, providerID, providerUserID string) (*AccountIdentity, error)
	UpdateAccountIdentity(ctx context.Context, identity *AccountIdentity) error
	UpdateIdentityLastUsed(ctx context.Context, identityID uuid.UUID, ts time.Time) error
	SetPrimaryIdentity(ctx context.Context, accountID, identityID uuid.UUID) error
	SoftDeleteAccountIdentity(ctx context.Context, identityID uuid.UUID, unlinkedAt time.Time) error
	DeleteAccountIdentity(ctx context.Context, identityID uuid.UUID) error

	// Account Session operations
	CreateAccountSession(ctx context.Context, session *AccountSession) error
	GetAccountSession(ctx context.Context, sessionID uuid.UUID) (*AccountSession, error)
	GetActiveSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]*AccountSession, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID, ts time.Time) error
	RevokeAccountSessions(ctx context.Context, accountID uuid.UUID, ts time.Time) error

	// Audit log operations
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogsByAccountID(ctx context.Context, accountID uuid.UUID, limit int32) ([]*AuditLog, error)

	// Complex queries
	FindAccountByEmail(ctx context.Context, email string) (*Account, error)
	FindAccountByEOA(ctx context.Context, eoaAddress string) (*Account, error)

	// Transaction support
	RunInTransaction(ctx context.Context, fn func(ctx context.Context, repo AccountRepository) error) error
}

type AppClientRepository interface {
	CreateAppClient(ctx context.Context, client *AppClient) error
	GetAppClient(ctx context.Context, id uuid.UUID) (*AppClient, error)
	GetAppClientByOAuth2ID(ctx context.Context, oauth2ID string) (*AppClient, error)
	ListAppClients(ctx context.Context) ([]*AppClient, error)
	UpdateAppClient(ctx context.Context, client *AppClient) error
	DeleteAppClient(ctx context.Context, id uuid.UUID) error
}

// MessageTemplateRepository defines database operations for message templates.
type MessageTemplateRepository interface {
	CreateMessageTemplate(ctx context.Context, tmpl *MessageTemplate) error
	GetMessageTemplate(ctx context.Context, id uuid.UUID) (*MessageTemplate, error)
	GetMessageTemplateByName(ctx context.Context, name string) (*MessageTemplate, error)
	ListMessageTemplates(ctx context.Context) ([]*MessageTemplate, error)
	UpdateMessageTemplate(ctx context.Context, tmpl *MessageTemplate) error
	DeleteMessageTemplate(ctx context.Context, id uuid.UUID) error
	CountAppClientsByTemplateID(ctx context.Context, templateID uuid.UUID) (int64, error)
}

// Domain models — decoupled from sqlc-generated types.

type AppClient struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	OAuth2ClientID     string     `json:"oauth2_client_id"`
	FrontendURL        string     `json:"frontend_url"`
	LoginPath          string     `json:"login_path"`
	LogoutURL          string     `json:"logout_url"`
	AllowedCORSOrigins []string   `json:"allowed_cors_origins"`
	JWTSecret          string     `json:"jwt_secret,omitempty"`
	MessageTemplateID  *uuid.UUID `json:"message_template_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MessageTemplate struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Protocol     string    `json:"protocol"`
	Statement    string    `json:"statement"`
	Domain       string    `json:"domain"`
	URI          string    `json:"uri"`
	ChainID      int       `json:"chain_id"`
	Version      string    `json:"version"`
	NonceTTLSecs int       `json:"nonce_ttl_secs"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Account struct {
	AccountID   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
	Status      string
	Metadata    []byte // raw JSONB
}

type IdentityProvider struct {
	ProviderID    string
	ProviderName  string
	ProviderType  string
	Enabled       bool
	Configuration []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AccountIdentity struct {
	IdentityID       uuid.UUID
	AccountID        uuid.UUID
	KratosIdentityID uuid.UUID
	ProviderID       string
	ProviderUserID   string
	DisplayName      *string
	AvatarURL        *string
	Attributes       []byte
	RawData          []byte
	Verified         bool
	IsPrimary        bool
	LinkedAt         time.Time
	LastUsedAt       *time.Time
	UpdatedAt        time.Time
	UnlinkedAt       *time.Time
}

type AccountSession struct {
	SessionID       uuid.UUID
	AccountID       uuid.UUID
	IdentityID      uuid.UUID
	KratosSessionID uuid.UUID
	IPAddress       *string
	UserAgent       *string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	LastActivityAt  *time.Time
}

type AuditLog struct {
	LogID           uuid.UUID
	AccountID       uuid.UUID
	IdentityID      *string
	EventType       string
	EventStatus     string
	EventMessage    *string
	SessionID       uuid.UUID
	KratosSessionID *string
	IPAddress       *string
	UserAgent       *string
	ProviderID      *string
	EventData       []byte
	CreatedAt       time.Time
}

// Status constants
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

// Provider type constants
const (
	ProviderTypeOAuth2 = "oauth2"
	ProviderTypeOIDC   = "oidc"
	ProviderTypeWeb3   = "web3"
	ProviderTypeEmail  = "email"
)

// Provider ID constants
const (
	ProviderGoogle   = "google"
	ProviderGithub   = "github"
	ProviderFacebook = "facebook"
	ProviderApple    = "apple"
	ProviderEmail    = "email"
	ProviderEOA      = "eoa"
)

// Audit event types
const (
	EventLogin            = "LOGIN"
	EventLogout           = "LOGOUT"
	EventRegistration     = "REGISTRATION"
	EventIdentityLinked   = "IDENTITY_LINKED"
	EventIdentityUnlinked = "IDENTITY_UNLINKED"
	EventSessionRevoked   = "SESSION_REVOKED"
)

const (
	EventStatusSuccess = "SUCCESS"
	EventStatusFailure = "FAILURE"
)

// UUIDToPgtype converts a google/uuid to pgtype.UUID for sqlc.
func UUIDToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

// PgtypeToUUID converts pgtype.UUID back to google/uuid.
func PgtypeToUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}
