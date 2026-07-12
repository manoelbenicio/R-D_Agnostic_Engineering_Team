package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

// dummyPasswordHash keeps the missing-user path on the same expensive bcrypt
// code path as a wrong password, reducing account-enumeration timing signals.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("invalid-credential"), bcrypt.DefaultCost)

// AuthIdentity is the provider-neutral link to a local Multica user.
type AuthIdentity struct {
	UserID pgtype.UUID
}

// AuthProvider isolates credential verification from HTTP/session handling.
// A Firebase adapter can implement this contract without changing /auth/login.
type AuthProvider interface {
	Login(ctx context.Context, email, password string) (AuthIdentity, error)
	Logout(ctx context.Context) error
}

type PasswordCredentialStore interface {
	FindByEmail(ctx context.Context, email string) (userID pgtype.UUID, passwordHash string, err error)
}

type PostgresPasswordCredentialStore struct {
	db dbExecutor
}

func NewPostgresPasswordCredentialStore(db dbExecutor) *PostgresPasswordCredentialStore {
	return &PostgresPasswordCredentialStore{db: db}
}

func (s *PostgresPasswordCredentialStore) FindByEmail(ctx context.Context, email string) (pgtype.UUID, string, error) {
	const query = `
		SELECT c.user_id, c.password_hash
		FROM user_password_credential c
		JOIN "user" u ON u.id = c.user_id
		WHERE lower(u.email) = lower($1)
	`
	var userID pgtype.UUID
	var passwordHash string
	err := s.db.QueryRow(ctx, query, email).Scan(&userID, &passwordHash)
	return userID, passwordHash, err
}

type PasswordAuthProvider struct {
	credentials PasswordCredentialStore
}

func NewPasswordAuthProvider(credentials PasswordCredentialStore) *PasswordAuthProvider {
	return &PasswordAuthProvider{credentials: credentials}
}

func (p *PasswordAuthProvider) Login(ctx context.Context, email, password string) (AuthIdentity, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return AuthIdentity{}, ErrInvalidCredentials
	}
	userID, passwordHash, err := p.credentials.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return AuthIdentity{}, ErrInvalidCredentials
		}
		return AuthIdentity{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return AuthIdentity{}, ErrInvalidCredentials
	}
	return AuthIdentity{UserID: userID}, nil
}

// Logout is stateless for the local provider; the handler clears the JWT
// cookies. Firebase can revoke provider-side state in its implementation.
func (p *PasswordAuthProvider) Logout(context.Context) error { return nil }

// TODO(native-runtimes-onboarding/1.7): implement credential provisioning only
// after the owner selects the operator seed/signup policy. Login must never
// create a credential or claim an existing account.
type PasswordCredentialProvisioner interface {
	ProvisionPassword(ctx context.Context, userID pgtype.UUID, password string) error
}
