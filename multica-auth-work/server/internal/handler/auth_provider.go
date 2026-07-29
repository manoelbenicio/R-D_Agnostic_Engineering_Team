package handler

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

var (
	ErrPasswordRequired        = errors.New("password is required")
	ErrPasswordTooShort        = errors.New("password must contain at least 12 characters")
	ErrPasswordTooLong         = errors.New("password exceeds bcrypt's 72-byte limit")
	ErrPasswordInvalidEncoding = errors.New("password must be valid UTF-8")
)

const (
	PasswordMinCharacters = 12
	PasswordMaxBytes      = 72
)

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

// ValidatePassword applies the server-authoritative local password policy.
// It intentionally avoids composition rules so long passphrases remain valid.
func ValidatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if !utf8.ValidString(password) {
		return ErrPasswordInvalidEncoding
	}
	if utf8.RuneCountInString(password) < PasswordMinCharacters {
		return ErrPasswordTooShort
	}
	if len(password) > PasswordMaxBytes {
		return ErrPasswordTooLong
	}
	return nil
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

func (s *PostgresPasswordCredentialStore) ProvisionPassword(ctx context.Context, userID pgtype.UUID, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	const query = `
		INSERT INTO user_password_credential (user_id, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
			updated_at = now()
	`
	_, err = s.db.Exec(ctx, query, userID, string(passwordHash))
	return err
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

type PasswordCredentialProvisioner interface {
	ProvisionPassword(ctx context.Context, userID pgtype.UUID, password string) error
}

var _ PasswordCredentialProvisioner = (*PostgresPasswordCredentialStore)(nil)
