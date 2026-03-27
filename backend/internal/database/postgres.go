package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/web3-lab/backend/internal/database/sqlc"
)

// PostgresRepository implements AccountRepository using sqlc-generated code.
type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// --- Conversion helpers ---

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

func fromUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func toTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func fromTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func strPtr(s string) *string { return &s }

// --- Domain ↔ sqlc conversions ---

func accountFromSqlc(a sqlc.Account) *Account {
	return &Account{
		AccountID:   fromUUID(a.AccountID),
		CreatedAt:   fromTimestamptz(a.CreatedAt),
		UpdatedAt:   fromTimestamptz(a.UpdatedAt),
		LastLoginAt: fromTimestamptzPtr(a.LastLoginAt),
		Status:      a.Status,
		Metadata:    a.Metadata,
	}
}

func identityFromSqlc(i sqlc.AccountIdentity) *AccountIdentity {
	return &AccountIdentity{
		IdentityID:       fromUUID(i.IdentityID),
		AccountID:        fromUUID(i.AccountID),
		KratosIdentityID: fromUUID(i.KratosIdentityID),
		ProviderID:       i.ProviderID,
		ProviderUserID:   i.ProviderUserID,
		DisplayName:      fromText(i.DisplayName),
		AvatarURL:        fromText(i.AvatarUrl),
		Attributes:       i.Attributes,
		RawData:          i.RawData,
		Verified:         i.Verified,
		IsPrimary:        i.IsPrimary,
		LinkedAt:         fromTimestamptz(i.LinkedAt),
		LastUsedAt:       fromTimestamptzPtr(i.LastUsedAt),
		UpdatedAt:        fromTimestamptz(i.UpdatedAt),
		UnlinkedAt:       fromTimestamptzPtr(i.UnlinkedAt),
	}
}

func sessionFromSqlc(s sqlc.AccountSession) *AccountSession {
	return &AccountSession{
		SessionID:       fromUUID(s.SessionID),
		AccountID:       fromUUID(s.AccountID),
		IdentityID:      fromUUID(s.IdentityID),
		KratosSessionID: fromUUID(s.KratosSessionID),
		IPAddress:       fromText(s.IpAddress),
		UserAgent:       fromText(s.UserAgent),
		CreatedAt:       fromTimestamptz(s.CreatedAt),
		ExpiresAt:       fromTimestamptz(s.ExpiresAt),
		RevokedAt:       fromTimestamptzPtr(s.RevokedAt),
		LastActivityAt:  fromTimestamptzPtr(s.LastActivityAt),
	}
}

func auditFromSqlc(a sqlc.AccountAuditLog) *AuditLog {
	return &AuditLog{
		LogID:           fromUUID(a.LogID),
		AccountID:       fromUUID(a.AccountID),
		IdentityID:      fromText(a.IdentityID),
		EventType:       a.EventType,
		EventStatus:     a.EventStatus,
		EventMessage:    fromText(a.EventMessage),
		SessionID:       fromUUID(a.SessionID),
		KratosSessionID: fromText(a.KratosSessionID),
		IPAddress:       fromText(a.IpAddress),
		UserAgent:       fromText(a.UserAgent),
		ProviderID:      fromText(a.ProviderID),
		EventData:       a.EventData,
		CreatedAt:       fromTimestamptz(a.CreatedAt),
	}
}

func appClientFromSqlc(c sqlc.AppClient) *AppClient {
	var origins []string
	if len(c.AllowedCorsOrigins) > 0 {
		_ = json.Unmarshal(c.AllowedCorsOrigins, &origins)
	}
	if origins == nil {
		origins = []string{}
	}

	var templateID *uuid.UUID
	if c.MessageTemplateID.Valid {
		id := fromUUID(c.MessageTemplateID)
		templateID = &id
	}

	return &AppClient{
		ID:                 fromUUID(c.ID),
		Name:               c.Name,
		OAuth2ClientID:     c.Oauth2ClientID,
		FrontendURL:        c.FrontendUrl,
		LoginPath:          c.LoginPath.String,
		LogoutURL:          c.LogoutUrl.String,
		AllowedCORSOrigins: origins,
		JWTSecret:          c.JwtSecret.String,
		MessageTemplateID:  templateID,
		CreatedAt:          fromTimestamptz(c.CreatedAt),
		UpdatedAt:          fromTimestamptz(c.UpdatedAt),
	}
}

func messageTemplateFromSqlc(m sqlc.MessageTemplate) *MessageTemplate {
	return &MessageTemplate{
		ID:           fromUUID(m.ID),
		Name:         m.Name,
		Protocol:     m.Protocol,
		Statement:    m.Statement,
		Domain:       m.Domain,
		URI:          m.Uri,
		ChainID:      int(m.ChainID),
		Version:      m.Version,
		NonceTTLSecs: int(m.NonceTtlSecs),
		CreatedAt:    fromTimestamptz(m.CreatedAt),
		UpdatedAt:    fromTimestamptz(m.UpdatedAt),
	}
}

// --- AccountRepository implementation ---

func (r *PostgresRepository) CreateAccount(ctx context.Context, acct *Account) error {
	result, err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		AccountID: toUUID(acct.AccountID),
		Status:    acct.Status,
		Metadata:  acct.Metadata,
	})
	if err != nil {
		return err
	}
	acct.CreatedAt = fromTimestamptz(result.CreatedAt)
	acct.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	a, err := r.queries.GetAccountByID(ctx, toUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return accountFromSqlc(a), nil
}

func (r *PostgresRepository) GetAccountByKratosIdentityID(ctx context.Context, kratosIdentityID uuid.UUID) (*Account, error) {
	a, err := r.queries.GetAccountByKratosIdentityID(ctx, toUUID(kratosIdentityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return accountFromSqlc(a), nil
}

func (r *PostgresRepository) UpdateAccount(ctx context.Context, acct *Account) error {
	return r.queries.UpdateAccount(ctx, sqlc.UpdateAccountParams{
		AccountID:   toUUID(acct.AccountID),
		LastLoginAt: toTimestamptzPtr(acct.LastLoginAt),
		Status:      acct.Status,
		Metadata:    acct.Metadata,
	})
}

func (r *PostgresRepository) UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status string) error {
	return r.queries.UpdateAccountStatus(ctx, sqlc.UpdateAccountStatusParams{
		AccountID: toUUID(accountID),
		Status:    status,
	})
}

func (r *PostgresRepository) UpdateLastLogin(ctx context.Context, accountID uuid.UUID, ts time.Time) error {
	return r.queries.UpdateLastLogin(ctx, sqlc.UpdateLastLoginParams{
		AccountID:   toUUID(accountID),
		LastLoginAt: toTimestamptz(ts),
	})
}

func (r *PostgresRepository) GetIdentityProvider(ctx context.Context, providerID string) (*IdentityProvider, error) {
	p, err := r.queries.GetIdentityProvider(ctx, providerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &IdentityProvider{
		ProviderID:    p.ProviderID,
		ProviderName:  p.ProviderName,
		ProviderType:  p.ProviderType,
		Enabled:       p.Enabled,
		Configuration: p.Configuration,
		CreatedAt:     fromTimestamptz(p.CreatedAt),
		UpdatedAt:     fromTimestamptz(p.UpdatedAt),
	}, nil
}

func (r *PostgresRepository) ListIdentityProviders(ctx context.Context, onlyEnabled bool) ([]*IdentityProvider, error) {
	rows, err := r.queries.ListIdentityProviders(ctx, onlyEnabled)
	if err != nil {
		return nil, err
	}
	result := make([]*IdentityProvider, 0, len(rows))
	for _, p := range rows {
		result = append(result, &IdentityProvider{
			ProviderID:    p.ProviderID,
			ProviderName:  p.ProviderName,
			ProviderType:  p.ProviderType,
			Enabled:       p.Enabled,
			Configuration: p.Configuration,
			CreatedAt:     fromTimestamptz(p.CreatedAt),
			UpdatedAt:     fromTimestamptz(p.UpdatedAt),
		})
	}
	return result, nil
}

func (r *PostgresRepository) CreateAccountIdentity(ctx context.Context, ident *AccountIdentity) error {
	result, err := r.queries.CreateAccountIdentity(ctx, sqlc.CreateAccountIdentityParams{
		IdentityID:       toUUID(ident.IdentityID),
		AccountID:        toUUID(ident.AccountID),
		KratosIdentityID: toUUID(ident.KratosIdentityID),
		ProviderID:       ident.ProviderID,
		ProviderUserID:   ident.ProviderUserID,
		DisplayName:      toText(ident.DisplayName),
		AvatarUrl:        toText(ident.AvatarURL),
		Attributes:       ident.Attributes,
		RawData:          ident.RawData,
		Verified:         ident.Verified,
		IsPrimary:        ident.IsPrimary,
	})
	if err != nil {
		return err
	}
	ident.LinkedAt = fromTimestamptz(result.LinkedAt)
	ident.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) GetAccountIdentity(ctx context.Context, identityID uuid.UUID) (*AccountIdentity, error) {
	i, err := r.queries.GetAccountIdentity(ctx, toUUID(identityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return identityFromSqlc(i), nil
}

func (r *PostgresRepository) GetAccountIdentityByKratosID(ctx context.Context, kratosIdentityID uuid.UUID) (*AccountIdentity, error) {
	i, err := r.queries.GetAccountIdentityByKratosID(ctx, toUUID(kratosIdentityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return identityFromSqlc(i), nil
}

func (r *PostgresRepository) GetAccountIdentityByProviderUserID(ctx context.Context, providerID, providerUserID string) (*AccountIdentity, error) {
	i, err := r.queries.GetAccountIdentityByProviderUserID(ctx, sqlc.GetAccountIdentityByProviderUserIDParams{
		ProviderID:     providerID,
		ProviderUserID: providerUserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return identityFromSqlc(i), nil
}

func (r *PostgresRepository) GetAccountIdentitiesByAccountID(ctx context.Context, accountID uuid.UUID) ([]*AccountIdentity, error) {
	rows, err := r.queries.GetAccountIdentitiesByAccountID(ctx, toUUID(accountID))
	if err != nil {
		return nil, err
	}
	result := make([]*AccountIdentity, 0, len(rows))
	for _, i := range rows {
		result = append(result, identityFromSqlc(i))
	}
	return result, nil
}

func (r *PostgresRepository) UpdateAccountIdentity(ctx context.Context, ident *AccountIdentity) error {
	return r.queries.UpdateAccountIdentity(ctx, sqlc.UpdateAccountIdentityParams{
		IdentityID:  toUUID(ident.IdentityID),
		AccountID:   toUUID(ident.AccountID),
		DisplayName: toText(ident.DisplayName),
		AvatarUrl:   toText(ident.AvatarURL),
		Attributes:  ident.Attributes,
		RawData:     ident.RawData,
		Verified:    ident.Verified,
		IsPrimary:   ident.IsPrimary,
		LastUsedAt:  toTimestamptzPtr(ident.LastUsedAt),
	})
}

func (r *PostgresRepository) UpdateIdentityLastUsed(ctx context.Context, identityID uuid.UUID, ts time.Time) error {
	return r.queries.UpdateIdentityLastUsed(ctx, sqlc.UpdateIdentityLastUsedParams{
		IdentityID: toUUID(identityID),
		LastUsedAt: toTimestamptz(ts),
	})
}

func (r *PostgresRepository) SetPrimaryIdentity(ctx context.Context, accountID, identityID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)
	if err := qtx.SetPrimaryIdentityReset(ctx, toUUID(accountID)); err != nil {
		return err
	}
	if err := qtx.SetPrimaryIdentitySet(ctx, sqlc.SetPrimaryIdentitySetParams{
		IdentityID: toUUID(identityID),
		AccountID:  toUUID(accountID),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) SoftDeleteAccountIdentity(ctx context.Context, identityID uuid.UUID, unlinkedAt time.Time) error {
	return r.queries.SoftDeleteAccountIdentity(ctx, sqlc.SoftDeleteAccountIdentityParams{
		IdentityID: toUUID(identityID),
		UnlinkedAt: toTimestamptz(unlinkedAt),
	})
}

func (r *PostgresRepository) CountActiveIdentitiesByAccountID(ctx context.Context, accountID uuid.UUID) (int64, error) {
	return r.queries.CountActiveIdentitiesByAccountID(ctx, toUUID(accountID))
}

func (r *PostgresRepository) DeleteAccountIdentity(ctx context.Context, identityID uuid.UUID) error {
	return r.queries.DeleteAccountIdentity(ctx, toUUID(identityID))
}

func (r *PostgresRepository) CreateAccountSession(ctx context.Context, sess *AccountSession) error {
	result, err := r.queries.CreateAccountSession(ctx, sqlc.CreateAccountSessionParams{
		SessionID:       toUUID(sess.SessionID),
		AccountID:       toUUID(sess.AccountID),
		IdentityID:      toUUID(sess.IdentityID),
		KratosSessionID: toUUID(sess.KratosSessionID),
		IpAddress:       toText(sess.IPAddress),
		UserAgent:       toText(sess.UserAgent),
		ExpiresAt:       toTimestamptz(sess.ExpiresAt),
	})
	if err != nil {
		return err
	}
	sess.CreatedAt = fromTimestamptz(result.CreatedAt)
	return nil
}

func (r *PostgresRepository) GetAccountSession(ctx context.Context, sessionID uuid.UUID) (*AccountSession, error) {
	s, err := r.queries.GetAccountSession(ctx, toUUID(sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sessionFromSqlc(s), nil
}

func (r *PostgresRepository) GetActiveSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]*AccountSession, error) {
	rows, err := r.queries.GetActiveSessionsByAccountID(ctx, toUUID(accountID))
	if err != nil {
		return nil, err
	}
	result := make([]*AccountSession, 0, len(rows))
	for _, s := range rows {
		result = append(result, sessionFromSqlc(s))
	}
	return result, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID, ts time.Time) error {
	return r.queries.RevokeSession(ctx, sqlc.RevokeSessionParams{
		SessionID: toUUID(sessionID),
		RevokedAt: toTimestamptz(ts),
	})
}

func (r *PostgresRepository) RevokeAccountSessions(ctx context.Context, accountID uuid.UUID, ts time.Time) error {
	return r.queries.RevokeAccountSessions(ctx, sqlc.RevokeAccountSessionsParams{
		AccountID: toUUID(accountID),
		RevokedAt: toTimestamptz(ts),
	})
}

func (r *PostgresRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	_, err := r.queries.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		LogID:           toUUID(log.LogID),
		AccountID:       toUUID(log.AccountID),
		IdentityID:      toText(log.IdentityID),
		EventType:       log.EventType,
		EventStatus:     log.EventStatus,
		EventMessage:    toText(log.EventMessage),
		SessionID:       toUUID(log.SessionID),
		KratosSessionID: toText(log.KratosSessionID),
		IpAddress:       toText(log.IPAddress),
		UserAgent:       toText(log.UserAgent),
		ProviderID:      toText(log.ProviderID),
		EventData:       log.EventData,
	})
	return err
}

func (r *PostgresRepository) GetAuditLogsByAccountID(ctx context.Context, accountID uuid.UUID, limit int32) ([]*AuditLog, error) {
	rows, err := r.queries.GetAuditLogsByAccountID(ctx, sqlc.GetAuditLogsByAccountIDParams{
		AccountID: toUUID(accountID),
		LogLimit:  limit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*AuditLog, 0, len(rows))
	for _, a := range rows {
		result = append(result, auditFromSqlc(a))
	}
	return result, nil
}

func (r *PostgresRepository) FindAccountByEmail(ctx context.Context, email string) (*Account, error) {
	a, err := r.queries.FindAccountByEmail(ctx, []byte(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return accountFromSqlc(a), nil
}

func (r *PostgresRepository) FindAccountByEOA(ctx context.Context, eoaAddress string) (*Account, error) {
	a, err := r.queries.FindAccountByEOA(ctx, []byte(eoaAddress))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return accountFromSqlc(a), nil
}

func (r *PostgresRepository) RunInTransaction(ctx context.Context, fn func(ctx context.Context, repo AccountRepository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := &PostgresRepository{
		pool:    r.pool, // keep pool ref for nested tx if needed
		queries: r.queries.WithTx(tx),
	}

	if err := fn(ctx, txRepo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// --- AppClientRepository implementation ---

func (r *PostgresRepository) CreateAppClient(ctx context.Context, client *AppClient) error {
	originsJSON, err := json.Marshal(client.AllowedCORSOrigins)
	if err != nil {
		return fmt.Errorf("marshal origins: %w", err)
	}

	result, err := r.queries.CreateAppClient(ctx, sqlc.CreateAppClientParams{
		ID:                 toUUID(client.ID),
		Name:               client.Name,
		Oauth2ClientID:     client.OAuth2ClientID,
		FrontendUrl:        client.FrontendURL,
		LoginPath:          pgtype.Text{String: client.LoginPath, Valid: client.LoginPath != ""},
		LogoutUrl:          pgtype.Text{String: client.LogoutURL, Valid: client.LogoutURL != ""},
		AllowedCorsOrigins: originsJSON,
		JwtSecret:          pgtype.Text{String: client.JWTSecret, Valid: client.JWTSecret != ""},
	})
	if err != nil {
		return err
	}
	client.CreatedAt = fromTimestamptz(result.CreatedAt)
	client.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) GetAppClient(ctx context.Context, id uuid.UUID) (*AppClient, error) {
	c, err := r.queries.GetAppClient(ctx, toUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return appClientFromSqlc(c), nil
}

func (r *PostgresRepository) GetAppClientByOAuth2ID(ctx context.Context, oauth2ID string) (*AppClient, error) {
	c, err := r.queries.GetAppClientByOAuth2ID(ctx, oauth2ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return appClientFromSqlc(c), nil
}

func (r *PostgresRepository) ListAppClients(ctx context.Context) ([]*AppClient, error) {
	rows, err := r.queries.ListAppClients(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*AppClient, 0, len(rows))
	for _, c := range rows {
		result = append(result, appClientFromSqlc(c))
	}
	return result, nil
}

func (r *PostgresRepository) UpdateAppClient(ctx context.Context, client *AppClient) error {
	originsJSON, err := json.Marshal(client.AllowedCORSOrigins)
	if err != nil {
		return fmt.Errorf("marshal origins: %w", err)
	}

	result, err := r.queries.UpdateAppClient(ctx, sqlc.UpdateAppClientParams{
		ID:                 toUUID(client.ID),
		Name:               client.Name,
		FrontendUrl:        client.FrontendURL,
		LoginPath:          pgtype.Text{String: client.LoginPath, Valid: client.LoginPath != ""},
		LogoutUrl:          pgtype.Text{String: client.LogoutURL, Valid: client.LogoutURL != ""},
		AllowedCorsOrigins: originsJSON,
		JwtSecret:          pgtype.Text{String: client.JWTSecret, Valid: client.JWTSecret != ""},
	})
	if err != nil {
		return err
	}
	client.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) DeleteAppClient(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteAppClient(ctx, toUUID(id))
}

// --- MessageTemplateRepository implementation ---

func (r *PostgresRepository) CreateMessageTemplate(ctx context.Context, tmpl *MessageTemplate) error {
	result, err := r.queries.CreateMessageTemplate(ctx, sqlc.CreateMessageTemplateParams{
		Name:         tmpl.Name,
		Protocol:     tmpl.Protocol,
		Statement:    tmpl.Statement,
		Domain:       tmpl.Domain,
		Uri:          tmpl.URI,
		ChainID:      int32(tmpl.ChainID),
		Version:      tmpl.Version,
		NonceTtlSecs: int32(tmpl.NonceTTLSecs),
	})
	if err != nil {
		return err
	}
	tmpl.ID = fromUUID(result.ID)
	tmpl.CreatedAt = fromTimestamptz(result.CreatedAt)
	tmpl.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) GetMessageTemplate(ctx context.Context, id uuid.UUID) (*MessageTemplate, error) {
	m, err := r.queries.GetMessageTemplate(ctx, toUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return messageTemplateFromSqlc(m), nil
}

func (r *PostgresRepository) GetMessageTemplateByName(ctx context.Context, name string) (*MessageTemplate, error) {
	m, err := r.queries.GetMessageTemplateByName(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return messageTemplateFromSqlc(m), nil
}

func (r *PostgresRepository) ListMessageTemplates(ctx context.Context) ([]*MessageTemplate, error) {
	rows, err := r.queries.ListMessageTemplates(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*MessageTemplate, 0, len(rows))
	for _, m := range rows {
		result = append(result, messageTemplateFromSqlc(m))
	}
	return result, nil
}

func (r *PostgresRepository) UpdateMessageTemplate(ctx context.Context, tmpl *MessageTemplate) error {
	result, err := r.queries.UpdateMessageTemplate(ctx, sqlc.UpdateMessageTemplateParams{
		ID:           toUUID(tmpl.ID),
		Column2:      tmpl.Name,
		Column3:      tmpl.Statement,
		Column4:      tmpl.Domain,
		Column5:      tmpl.URI,
		ChainID:      int32(tmpl.ChainID),
		Column7:      tmpl.Version,
		NonceTtlSecs: int32(tmpl.NonceTTLSecs),
	})
	if err != nil {
		return err
	}
	tmpl.UpdatedAt = fromTimestamptz(result.UpdatedAt)
	return nil
}

func (r *PostgresRepository) DeleteMessageTemplate(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteMessageTemplate(ctx, toUUID(id))
}

func (r *PostgresRepository) CountAppClientsByTemplateID(ctx context.Context, templateID uuid.UUID) (int64, error) {
	return r.queries.CountAppClientsByTemplateID(ctx, toUUID(templateID))
}

