// Package passwordtest isolates the stub-based password provisioning and
// update unit tests from the DB-gated TestMain in package handler.
//
// The parent package's TestMain (handler_test.go) calls os.Exit(0) when
// PostgreSQL is unreachable, which would silence these assertions offline
// even though they exercise in-memory stubs and never touch a database.
// Living in a separate package with no TestMain of its own, they run under
// the default test runner whenever `go test ./internal/handler/passwordtest/`
// is invoked, with or without PostgreSQL.
//
// The tests reuse the exported handler surface plus the unexported
// handler.dbExecutor interface, which the local stub satisfies structurally
// (Go interface satisfaction is independent of export status, so a value
// whose methods match an unexported interface may be passed to a function
// that accepts that interface without naming it).
package passwordtest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type passwordProvisionDB struct {
	query string
	args  []any
	err   error
}

func (d *passwordProvisionDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	d.query = query
	d.args = args
	return pgconn.CommandTag{}, d.err
}

func (*passwordProvisionDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (*passwordProvisionDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected query row")
}

func TestPostgresPasswordCredentialStoreProvisionPasswordUsesDefaultCostUpsert(t *testing.T) {
	database := &passwordProvisionDB{}
	store := handler.NewPostgresPasswordCredentialStore(database)
	userID := pgtype.UUID{Bytes: [16]byte{9}, Valid: true}
	password := " synthetic password with spaces "

	if err := store.ProvisionPassword(context.Background(), userID, password); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.query, "ON CONFLICT (user_id) DO UPDATE") ||
		!strings.Contains(database.query, "password_hash = EXCLUDED.password_hash") ||
		!strings.Contains(database.query, "updated_at = now()") {
		t.Fatalf("query does not perform the required timestamped upsert: %s", database.query)
	}
	if len(database.args) != 2 || database.args[0] != userID {
		t.Fatalf("args = %#v", database.args)
	}
	hash, ok := database.args[1].(string)
	if !ok {
		t.Fatalf("password hash arg type = %T", database.args[1])
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatal("stored hash does not match the untrimmed synthetic password")
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != bcrypt.DefaultCost {
		t.Fatalf("bcrypt cost = %d, err = %v, want %d", cost, err, bcrypt.DefaultCost)
	}
}

func TestPostgresPasswordCredentialStoreProvisionPasswordValidatesBeforeWrite(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "empty", password: "", wantErr: handler.ErrPasswordRequired},
		{name: "too short", password: "short-value", wantErr: handler.ErrPasswordTooShort},
		{name: "over bcrypt limit", password: strings.Repeat("x", 73), wantErr: handler.ErrPasswordTooLong},
		{name: "invalid UTF-8", password: string([]byte{0xff, 0xfe}), wantErr: handler.ErrPasswordInvalidEncoding},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := &passwordProvisionDB{}
			err := handler.NewPostgresPasswordCredentialStore(database).ProvisionPassword(context.Background(), pgtype.UUID{Valid: true}, tc.password)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if database.query != "" {
				t.Fatal("invalid password reached the database")
			}
		})
	}
}

type passwordProvisionerStub struct {
	userID   pgtype.UUID
	password string
	err      error
	called   bool
}

func (p *passwordProvisionerStub) ProvisionPassword(_ context.Context, userID pgtype.UUID, password string) error {
	p.called = true
	p.userID = userID
	p.password = password
	return p.err
}

type passwordUserDB struct {
	user db.User
}

func (*passwordUserDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*passwordUserDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (d *passwordUserDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return passwordUserRow{user: d.user}
}

type passwordUserRow struct {
	user db.User
}

func (r passwordUserRow) Scan(dest ...any) error {
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

type passwordAuthProviderStub struct {
	identity handler.AuthIdentity
	err      error
	email    string
	password string
}

func (p *passwordAuthProviderStub) Login(_ context.Context, email, password string) (handler.AuthIdentity, error) {
	p.email = email
	p.password = password
	return p.identity, p.err
}

func (*passwordAuthProviderStub) Logout(context.Context) error { return nil }

func TestUpdatePasswordRequiresCurrentPasswordWithoutRecentJWT(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	userIDString := util.UUIDToString(userID)
	provider := &passwordAuthProviderStub{identity: handler.AuthIdentity{UserID: userID}}
	provisioner := &passwordProvisionerStub{}
	h := &handler.Handler{
		AuthProvider:        provider,
		PasswordProvisioner: provisioner,
		Queries:             db.New(&passwordUserDB{user: db.User{ID: userID, Name: "User", Email: "user@example.com"}}),
	}
	req := httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(`{"current_password":"current-secret","new_password":"replacement-secret"}`))
	req.Header.Set("X-User-ID", userIDString)
	res := httptest.NewRecorder()

	h.UpdatePassword(res, req)

	if res.Code != http.StatusNoContent || !provisioner.called {
		t.Fatalf("status = %d, provisioned = %v, body = %s", res.Code, provisioner.called, res.Body.String())
	}
	if provider.email != "user@example.com" || provider.password != "current-secret" {
		t.Fatal("current password was not verified against the authenticated local user")
	}
}

func TestUpdatePasswordRejectsMissingOrInvalidCurrentAuthentication(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	for _, tc := range []struct {
		name     string
		current  string
		provider *passwordAuthProviderStub
	}{
		{name: "missing"},
		{name: "invalid", current: "incorrect-secret", provider: &passwordAuthProviderStub{err: handler.ErrInvalidCredentials}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provisioner := &passwordProvisionerStub{}
			h := &handler.Handler{
				AuthProvider:        tc.provider,
				PasswordProvisioner: provisioner,
				Queries:             db.New(&passwordUserDB{user: db.User{ID: userID, Name: "User", Email: "user@example.com"}}),
			}
			body := `{"new_password":"replacement-secret"}`
			if tc.current != "" {
				body = `{"current_password":"` + tc.current + `","new_password":"replacement-secret"}`
			}
			req := httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(body))
			req.Header.Set("X-User-ID", util.UUIDToString(userID))
			res := httptest.NewRecorder()

			h.UpdatePassword(res, req)

			if res.Code != http.StatusUnauthorized || provisioner.called {
				t.Fatalf("status = %d, provisioned = %v, body = %s", res.Code, provisioner.called, res.Body.String())
			}
			if tc.current != "" && strings.Contains(res.Body.String(), tc.current) {
				t.Fatal("response exposed current password")
			}
		})
	}
}

func TestPasswordLoginMintsRecentAuthProofButCLITokenDoesNot(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}
	user := db.User{ID: userID, Name: "User", Email: "user@example.com", OnboardingQuestionnaire: []byte(`{}`)}
	queries := db.New(&passwordUserDB{user: user})
	h := &handler.Handler{
		AuthProvider: &passwordAuthProviderStub{identity: handler.AuthIdentity{UserID: userID}},
		Queries:      queries,
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"current-secret"}`))
	loginRes := httptest.NewRecorder()
	h.Login(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginRes.Code, loginRes.Body.String())
	}
	var loginResponse handler.LoginResponse
	if err := json.Unmarshal(loginRes.Body.Bytes(), &loginResponse); err != nil {
		t.Fatal(err)
	}
	if _, ok := parseJWTClaims(t, loginResponse.Token)["auth_time"]; !ok {
		t.Fatal("password login token did not carry recent-auth proof")
	}

	cliReq := httptest.NewRequest(http.MethodPost, "/api/cli-token", nil)
	cliReq.Header.Set("X-User-ID", util.UUIDToString(userID))
	cliRes := httptest.NewRecorder()
	h.IssueCliToken(cliRes, cliReq)
	if cliRes.Code != http.StatusOK {
		t.Fatalf("CLI token status = %d, body = %s", cliRes.Code, cliRes.Body.String())
	}
	var cliResponse map[string]string
	if err := json.Unmarshal(cliRes.Body.Bytes(), &cliResponse); err != nil {
		t.Fatal(err)
	}
	if _, ok := parseJWTClaims(t, cliResponse["token"])["auth_time"]; ok {
		t.Fatal("CLI token exchange manufactured recent-auth proof")
	}
}

func parseJWTClaims(t *testing.T, raw string) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(raw, func(*jwt.Token) (any, error) { return auth.JWTSecret(), nil })
	if err != nil || !token.Valid {
		t.Fatalf("parse JWT: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("JWT claims type mismatch")
	}
	return claims
}

func TestUpdatePasswordUsesAuthenticatedUserAndPreservesWhitespace(t *testing.T) {
	provisioner := &passwordProvisionerStub{}
	h := &handler.Handler{PasswordProvisioner: provisioner}
	userID := "01972f7e-7e8d-77ef-a13d-1b0ce3e9c001"
	req := httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(`{"new_password":"  synthetic value  "}`))
	req = req.WithContext(auth.WithAuthenticationTime(req.Context(), time.Now()))
	req.Header.Set("X-User-ID", userID)
	res := httptest.NewRecorder()

	h.UpdatePassword(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if util.UUIDToString(provisioner.userID) != userID || provisioner.password != "  synthetic value  " {
		t.Fatalf("user=%s password whitespace was not preserved", util.UUIDToString(provisioner.userID))
	}
	if res.Body.Len() != 0 {
		t.Fatal("password update response must not echo a payload")
	}
}

func TestUpdatePasswordStrictRequestAndFailureHandling(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		provisioner *passwordProvisionerStub
		wantStatus  int
	}{
		{name: "missing authentication", body: `{"new_password":"synthetic-password"}`, provisioner: &passwordProvisionerStub{}, wantStatus: http.StatusUnauthorized},
		{name: "unknown field", body: `{"new_password":"synthetic-password","admin":true}`, provisioner: &passwordProvisionerStub{}, wantStatus: http.StatusBadRequest},
		{name: "trailing document", body: `{"new_password":"synthetic-password"} {}`, provisioner: &passwordProvisionerStub{}, wantStatus: http.StatusBadRequest},
		{name: "empty", body: `{"new_password":""}`, provisioner: &passwordProvisionerStub{err: handler.ErrPasswordRequired}, wantStatus: http.StatusBadRequest},
		{name: "too short", body: `{"new_password":"short-value"}`, provisioner: &passwordProvisionerStub{}, wantStatus: http.StatusBadRequest},
		{name: "store failure", body: `{"new_password":"synthetic-password"}`, provisioner: &passwordProvisionerStub{err: errors.New("database detail")}, wantStatus: http.StatusInternalServerError},
		{name: "not configured", body: `{"new_password":"synthetic-password"}`, wantStatus: http.StatusServiceUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &handler.Handler{}
			if tc.provisioner != nil {
				h.PasswordProvisioner = tc.provisioner
			}
			req := httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(tc.body))
			req = req.WithContext(auth.WithAuthenticationTime(req.Context(), time.Now()))
			if tc.name != "missing authentication" {
				req.Header.Set("X-User-ID", "01972f7e-7e8d-77ef-a13d-1b0ce3e9c001")
			}
			res := httptest.NewRecorder()
			h.UpdatePassword(res, req)
			if res.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
			if strings.Contains(res.Body.String(), "synthetic") || strings.Contains(res.Body.String(), "database detail") {
				t.Fatal("response exposed password or backend details")
			}
		})
	}
}

func TestUpdatePasswordRejectsOversizedBody(t *testing.T) {
	// 2048 bytes comfortably exceeds the 1 KiB MaxBytesReader cap that
	// handler.UpdatePassword applies via the unexported
	// passwordUpdateBodyLimit constant; any value over the cap triggers
	// the same 400, so the exact limit need not be mirrored here.
	h := &handler.Handler{PasswordProvisioner: &passwordProvisionerStub{}}
	body := `{"new_password":"` + strings.Repeat("x", 2048) + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/me/password", strings.NewReader(body))
	req = req.WithContext(auth.WithAuthenticationTime(req.Context(), time.Now()))
	req.Header.Set("X-User-ID", "01972f7e-7e8d-77ef-a13d-1b0ce3e9c001")
	res := httptest.NewRecorder()
	h.UpdatePassword(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
