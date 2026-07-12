package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/auth"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type authProviderStub struct {
	identity AuthIdentity
	loginErr error
	logout   bool
	email    string
	password string
}

type loginUserDB struct{ user db.User }

func (d *loginUserDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (d *loginUserDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (d *loginUserDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return loginUserRow{user: d.user}
}

type loginUserRow struct{ user db.User }

func (r loginUserRow) Scan(dest ...any) error {
	values := []any{
		r.user.ID, r.user.Name, r.user.Email, r.user.AvatarUrl, r.user.CreatedAt,
		r.user.UpdatedAt, r.user.OnboardedAt, r.user.OnboardingQuestionnaire,
		r.user.CloudWaitlistEmail, r.user.CloudWaitlistReason, r.user.StarterContentState,
		r.user.Language, r.user.ProfileDescription, r.user.Timezone,
	}
	for i, value := range values {
		switch target := dest[i].(type) {
		case *pgtype.UUID:
			*target = value.(pgtype.UUID)
		case *string:
			*target = value.(string)
		case *pgtype.Text:
			*target = value.(pgtype.Text)
		case *pgtype.Timestamptz:
			*target = value.(pgtype.Timestamptz)
		case *[]byte:
			*target = value.([]byte)
		}
	}
	return nil
}

func (p *authProviderStub) Login(_ context.Context, email, password string) (AuthIdentity, error) {
	p.email, p.password = email, password
	return p.identity, p.loginErr
}

func (p *authProviderStub) Logout(context.Context) error {
	p.logout = true
	return nil
}

func TestPasswordLoginInvalidCredentialIsUnauthorized(t *testing.T) {
	provider := &authProviderStub{loginErr: ErrInvalidCredentials}
	h := &Handler{AuthProvider: provider}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"not-logged"}`))
	res := httptest.NewRecorder()

	h.Login(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "not-logged") {
		t.Fatal("response leaked password")
	}
}

func TestPasswordLoginReturnsTokenUserAndSessionCookie(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	provider := &authProviderStub{identity: AuthIdentity{UserID: userID}}
	user := db.User{ID: userID, Name: "User", Email: "user@example.com", OnboardingQuestionnaire: []byte(`{}`)}
	h := &Handler{AuthProvider: provider, Queries: db.New(&loginUserDB{user: user})}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"correct"}`))
	res := httptest.NewRecorder()

	h.Login(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var response LoginResponse
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Token == "" || response.User.Email != user.Email {
		t.Fatalf("response=%#v", response)
	}
	if provider.password != "correct" {
		t.Fatal("provider did not receive password")
	}
	found := false
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == auth.AuthCookieName && cookie.Value != "" && cookie.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Fatal("HttpOnly session cookie missing")
	}
}

func TestPasswordLoginRejectsUnknownFields(t *testing.T) {
	h := &Handler{AuthProvider: &authProviderStub{}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"x","admin":true}`))
	res := httptest.NewRecorder()
	h.Login(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLogoutCallsProviderAndClearsSessionCookie(t *testing.T) {
	provider := &authProviderStub{}
	h := &Handler{AuthProvider: provider}
	res := httptest.NewRecorder()
	h.Logout(res, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if res.Code != http.StatusOK || !provider.logout {
		t.Fatalf("status=%d logout=%v", res.Code, provider.logout)
	}
	found := false
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == auth.AuthCookieName && cookie.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("auth session cookie was not cleared")
	}
}

func TestPasswordLoginProviderFailureIsGeneric(t *testing.T) {
	h := &Handler{AuthProvider: &authProviderStub{loginErr: errors.New("database details")}}
	res := httptest.NewRecorder()
	h.Login(res, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"secret"}`)))
	if res.Code != http.StatusInternalServerError || strings.Contains(res.Body.String(), "database details") {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestPasswordLoginRequestContract(t *testing.T) {
	var request PasswordLoginRequest
	if err := json.Unmarshal([]byte(`{"email":"user@example.com","password":"secret"}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.Email != "user@example.com" || request.Password != "secret" {
		t.Fatalf("request=%#v", request)
	}
}
