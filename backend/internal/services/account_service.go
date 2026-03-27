package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/web3-lab/backend/internal/database"
)

// AccountService provides business logic for accounts and identities.
type AccountService struct {
	repo database.AccountRepository
}

func NewAccountService(repo database.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateAccountWithIdentity creates a new account and links the initial identity in one transaction.
func (s *AccountService) CreateAccountWithIdentity(ctx context.Context, kratosUUID uuid.UUID, providerID, providerUserID string, attributes []byte) (*database.Account, *database.AccountIdentity, error) {
	var acct *database.Account
	var ident *database.AccountIdentity

	err := s.repo.RunInTransaction(ctx, func(ctx context.Context, txRepo database.AccountRepository) error {
		// Create account (no kratos_uuid — grouping only)
		acct = &database.Account{
			AccountID: uuid.New(),
			Status:    database.StatusActive,
		}
		if err := txRepo.CreateAccount(ctx, acct); err != nil {
			return fmt.Errorf("create account: %w", err)
		}

		// Create identity link
		ident = &database.AccountIdentity{
			IdentityID:       uuid.New(),
			AccountID:        acct.AccountID,
			KratosIdentityID: kratosUUID,
			ProviderID:       providerID,
			ProviderUserID:   providerUserID,
			Attributes:       attributes,
			Verified:         true,
			IsPrimary:        true,
			LinkedAt:         time.Now(),
		}
		if err := txRepo.CreateAccountIdentity(ctx, ident); err != nil {
			return fmt.Errorf("create identity: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return acct, ident, nil
}

func (s *AccountService) GetAccountByID(ctx context.Context, id uuid.UUID) (*database.Account, error) {
	return s.repo.GetAccountByID(ctx, id)
}

func (s *AccountService) GetAccountByKratosIdentityID(ctx context.Context, kratosIdentityID uuid.UUID) (*database.Account, error) {
	return s.repo.GetAccountByKratosIdentityID(ctx, kratosIdentityID)
}

func (s *AccountService) FindAccountByEOA(ctx context.Context, eoa string) (*database.Account, error) {
	return s.repo.FindAccountByEOA(ctx, eoa)
}

func (s *AccountService) GetIdentitiesByAccountID(ctx context.Context, accountID uuid.UUID) ([]*database.AccountIdentity, error) {
	return s.repo.GetAccountIdentitiesByAccountID(ctx, accountID)
}

func (s *AccountService) GetIdentityByProviderUserID(ctx context.Context, providerID, providerUserID string) (*database.AccountIdentity, error) {
	return s.repo.GetAccountIdentityByProviderUserID(ctx, providerID, providerUserID)
}

func (s *AccountService) GetAccountIdentityByKratosID(ctx context.Context, kratosIdentityID uuid.UUID) (*database.AccountIdentity, error) {
	return s.repo.GetAccountIdentityByKratosID(ctx, kratosIdentityID)
}

func (s *AccountService) UpdateLastLogin(ctx context.Context, accountID uuid.UUID) error {
	return s.repo.UpdateLastLogin(ctx, accountID, time.Now())
}

func (s *AccountService) SoftDeleteIdentity(ctx context.Context, identityID uuid.UUID) error {
	return s.repo.SoftDeleteAccountIdentity(ctx, identityID, time.Now())
}

func (s *AccountService) CreateAuditLog(ctx context.Context, log *database.AuditLog) error {
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *AccountService) GetActiveSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]*database.AccountSession, error) {
	return s.repo.GetActiveSessionsByAccountID(ctx, accountID)
}

func (s *AccountService) CreateSession(ctx context.Context, session *database.AccountSession) error {
	return s.repo.CreateAccountSession(ctx, session)
}

func (s *AccountService) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.RevokeSession(ctx, sessionID, time.Now())
}

// CountActiveIdentities returns the number of active (non-unlinked) identities for an account.
func (s *AccountService) CountActiveIdentities(ctx context.Context, accountID uuid.UUID) (int64, error) {
	return s.repo.CountActiveIdentitiesByAccountID(ctx, accountID)
}

// LinkIdentityToAccount creates a new (non-primary) identity linked to an existing account.
func (s *AccountService) LinkIdentityToAccount(ctx context.Context, accountID uuid.UUID, kratosUUID uuid.UUID, providerID, providerUserID string, attributes []byte) (*database.AccountIdentity, error) {
	ident := &database.AccountIdentity{
		IdentityID:       uuid.New(),
		AccountID:        accountID,
		KratosIdentityID: kratosUUID,
		ProviderID:       providerID,
		ProviderUserID:   providerUserID,
		Attributes:       attributes,
		Verified:         true,
		IsPrimary:        false,
		LinkedAt:         time.Now(),
	}
	if err := s.repo.CreateAccountIdentity(ctx, ident); err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}
	return ident, nil
}

// SafeUnlinkIdentity soft-deletes an identity only if it is not the last active identity for the account.
func (s *AccountService) SafeUnlinkIdentity(ctx context.Context, accountID, identityID uuid.UUID) error {
	count, err := s.repo.CountActiveIdentitiesByAccountID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("count identities: %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot unlink: this is the only remaining identity")
	}

	// Verify the identity belongs to this account
	ident, err := s.repo.GetAccountIdentity(ctx, identityID)
	if err != nil {
		return fmt.Errorf("get identity: %w", err)
	}
	if ident == nil {
		return fmt.Errorf("identity not found")
	}
	if ident.AccountID != accountID {
		return fmt.Errorf("identity does not belong to this account")
	}

	return s.repo.SoftDeleteAccountIdentity(ctx, identityID, time.Now())
}
