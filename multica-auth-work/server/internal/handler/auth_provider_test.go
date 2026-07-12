package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type credentialStoreStub struct {
	userID pgtype.UUID
	hash   string
	err    error
	email  string
}

func (s *credentialStoreStub) FindByEmail(_ context.Context, email string) (pgtype.UUID, string, error) {
	s.email = email
	return s.userID, s.hash, s.err
}

func TestPasswordAuthProviderLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	wantID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	store := &credentialStoreStub{userID: wantID, hash: string(hash)}
	provider := NewPasswordAuthProvider(store)

	identity, err := provider.Login(context.Background(), " User@Example.COM ", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if identity.UserID != wantID || store.email != "user@example.com" {
		t.Fatalf("identity=%#v email=%q", identity, store.email)
	}
}

func TestPasswordAuthProviderRejectsInvalidOrMissingCredential(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("right-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		store *credentialStoreStub
		pass  string
	}{
		{name: "wrong password", store: &credentialStoreStub{hash: string(hash)}, pass: "wrong-password"},
		{name: "no credential", store: &credentialStoreStub{err: pgx.ErrNoRows}, pass: "any-password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPasswordAuthProvider(tt.store).Login(context.Background(), "user@example.com", tt.pass)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPasswordAuthProviderPreservesStoreFailure(t *testing.T) {
	want := errors.New("database unavailable")
	_, err := NewPasswordAuthProvider(&credentialStoreStub{err: want}).Login(context.Background(), "user@example.com", "password")
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}
