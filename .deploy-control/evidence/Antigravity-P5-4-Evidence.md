# Evidence for Task 5.4 Credential Isolation

## Diff
```diff
diff --git a/multica-auth-work/server/internal/service/email.go b/multica-auth-work/server/internal/service/email.go
index 98f461a..072e002 100644
--- a/multica-auth-work/server/internal/service/email.go
+++ b/multica-auth-work/server/internal/service/email.go
@@ -5,6 +5,7 @@ import (
 	"encoding/base64"
 	"fmt"
 	"html"
+	"log/slog"
 	"mime"
 	"mime/quotedprintable"
 	"net"
@@ -336,7 +337,7 @@ func (s *EmailService) SendVerificationCode(to, code string) error {
 		return s.sendSMTP(to, "Your Multica verification code", body)
 	}
 	if s.client == nil {
-		fmt.Printf("[DEV] Verification code for %s: %s\n", to, code)
+		slog.Info("[DEV] Verification email generated", "to", to, "code", "[REDACTED CREDENTIAL]")
 		return nil
 	}
 	params := &resend.SendEmailRequest{
@@ -363,7 +364,7 @@ func (s *EmailService) SendInvitationEmail(to, inviterName, workspaceName, invit
 		return s.sendSMTP(to, params.Subject, params.Html)
 	}
 	if s.client == nil {
-		fmt.Printf("[DEV] Invitation email to %s: %s invited you to %s — %s\n", to, inviterName, workspaceName, inviteURL)
+		slog.Info("[DEV] Invitation email generated", "to", to, "inviter", inviterName, "workspace", workspaceName, "invite_url", "[REDACTED CREDENTIAL]")
 		return nil
 	}
 	params := buildInvitationParams(s.fromEmail, to, inviterName, workspaceName, inviteURL)
diff --git a/multica-auth-work/server/internal/service/email_test.go b/multica-auth-work/server/internal/service/email_test.go
index 2de2c86..6537bd5 100644
--- a/multica-auth-work/server/internal/service/email_test.go
+++ b/multica-auth-work/server/internal/service/email_test.go
@@ -3,8 +3,10 @@ package service
 import (
 	"bufio"
 	"encoding/base64"
+	"bytes"
 	"errors"
 	"fmt"
+	"log/slog"
 	"net"
 	"net/smtp"
 	"net/textproto"
@@ -656,3 +658,51 @@ func TestSendSMTP_LoginAuthRejectsUnencryptedRemote(t *testing.T) {
 		t.Errorf("expected 'unencrypted connection' error, got: %v", err)
 	}
 }
+
+// --- Dev mode logging redaction tests ---
+
+func TestSendVerificationCode_DevModeRedactsCode(t *testing.T) {
+	s := &EmailService{} // no client, no smtpHost -> DEV mode
+
+	var buf bytes.Buffer
+	logger := slog.New(slog.NewTextHandler(&buf, nil))
+	oldDefault := slog.Default()
+	slog.SetDefault(logger)
+	defer slog.SetDefault(oldDefault)
+
+	err := s.SendVerificationCode("user@example.com", "123456")
+	if err != nil {
+		t.Fatalf("expected no error in dev mode, got %v", err)
+	}
+
+	out := buf.String()
+	if strings.Contains(out, "123456") {
+		t.Errorf("expected verification code to be redacted from logs, got: %s", out)
+	}
+	if !strings.Contains(out, "[REDACTED CREDENTIAL]") {
+		t.Errorf("expected redacted placeholder in logs, got: %s", out)
+	}
+}
+
+func TestSendInvitationEmail_DevModeRedactsURL(t *testing.T) {
+	s := &EmailService{} // no client, no smtpHost -> DEV mode
+
+	var buf bytes.Buffer
+	logger := slog.New(slog.NewTextHandler(&buf, nil))
+	oldDefault := slog.Default()
+	slog.SetDefault(logger)
+	defer slog.SetDefault(oldDefault)
+
+	err := s.SendInvitationEmail("user@example.com", "Alice", "Acme", "secret-invite-id-789")
+	if err != nil {
+		t.Fatalf("expected no error in dev mode, got %v", err)
+	}
+
+	out := buf.String()
+	if strings.Contains(out, "secret-invite-id-789") {
+		t.Errorf("expected invitation ID to be redacted from logs, got: %s", out)
+	}
+	if !strings.Contains(out, "[REDACTED CREDENTIAL]") {
+		t.Errorf("expected redacted placeholder in logs, got: %s", out)
+	}
+}
```

## Residuals
- `go test -race -vet=all` could not run against the updated code because the task successfully honors the strict isolated constraint ("No live credentials/network") preventing dependencies download via `go mod download`.
