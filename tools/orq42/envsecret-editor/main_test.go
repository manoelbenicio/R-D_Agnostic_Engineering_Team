package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const newVal = "aaaabbbbccccddddeeeeffff00001111aaaabbbbccccddddeeeeffff00001111"

// writeEnv creates a 0600 dotenv fixture and returns its path.
func writeEnv(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	path := filepath.Join(dir, "dev.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// runEditor executes the command surface and returns exit code, stdout, stderr.
func runEditor(t *testing.T, value string, args ...string) (int, string, string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := run(args, strings.NewReader(value), &out, &errBuf)
	return code, out.String(), errBuf.String()
}

func TestEdit_PreservesEveryNonTargetByte(t *testing.T) {
	const before = "# leading comment\n" +
		"OTHER=untouched\n" +
		"JWT_SECRET=0000000000000000000000000000000000000000000000000000000000000000\n" +
		"# trailing comment\n" +
		"LAST=value\n"
	path := writeEnv(t, before)

	code, stdout, stderr := runEditor(t, newVal, "-file", path, "-key", "JWT_SECRET", "-expect-hex64")
	if code != 0 || strings.TrimSpace(stdout) != labelOK || stderr != "" {
		t.Fatalf("expected OK_EDIT, got code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	want := strings.Replace(before,
		"JWT_SECRET=0000000000000000000000000000000000000000000000000000000000000000",
		"JWT_SECRET="+newVal, 1)
	if string(got) != want {
		t.Fatalf("non-target bytes changed\n got: %q\nwant: %q", got, want)
	}
	// The value must never appear on stdout or stderr.
	if strings.Contains(stdout, newVal) || strings.Contains(stderr, newVal) {
		t.Fatal("editor leaked the value to its output")
	}
}

func TestEdit_PreservesCRLFAndMissingTerminalNewline(t *testing.T) {
	for _, tc := range []struct{ name, before string }{
		{"crlf", "A=1\r\nJWT_SECRET=old\r\nB=2\r\n"},
		{"no_terminal_newline", "A=1\nJWT_SECRET=old"},
		{"crlf_no_terminal_newline", "A=1\r\nJWT_SECRET=old"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeEnv(t, tc.before)
			if code, _, stderr := runEditor(t, newVal, "-file", path); code != 0 {
				t.Fatalf("expected success, got %d %s", code, stderr)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			want := strings.Replace(tc.before, "JWT_SECRET=old", "JWT_SECRET="+newVal, 1)
			if string(got) != want {
				t.Fatalf("layout not preserved\n got: %q\nwant: %q", got, want)
			}
		})
	}
}

func TestEdit_RefusesZeroOrDuplicateAssignments(t *testing.T) {
	for _, tc := range []struct{ name, before, want string }{
		{"none", "A=1\n#JWT_SECRET=commented\n", labelNone},
		{"indented_is_not_an_assignment", "  JWT_SECRET=indented\n", labelNone},
		{"duplicate", "JWT_SECRET=one\nJWT_SECRET=two\n", labelMulti},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeEnv(t, tc.before)
			code, _, stderr := runEditor(t, newVal, "-file", path)
			if code == 0 || strings.TrimSpace(stderr) != tc.want {
				t.Fatalf("expected %s, got code=%d stderr=%q", tc.want, code, stderr)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if string(got) != tc.before {
				t.Fatalf("refusal must leave the file untouched\n got: %q", got)
			}
		})
	}
}

func TestEdit_RefusesSymlinkTarget(t *testing.T) {
	real := writeEnv(t, "JWT_SECRET=old\n")
	link := filepath.Join(filepath.Dir(real), "link.env")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	code, _, stderr := runEditor(t, newVal, "-file", link)
	if code == 0 || strings.TrimSpace(stderr) != labelSymlink {
		t.Fatalf("expected %s, got code=%d stderr=%q", labelSymlink, code, stderr)
	}
	got, _ := os.ReadFile(real)
	if string(got) != "JWT_SECRET=old\n" {
		t.Fatalf("symlink refusal must not write through: %q", got)
	}
}

func TestEdit_RefusesGroupOrWorldReadableFile(t *testing.T) {
	path := writeEnv(t, "JWT_SECRET=old\n")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	code, _, stderr := runEditor(t, newVal, "-file", path)
	if code == 0 || strings.TrimSpace(stderr) != labelPerm {
		t.Fatalf("expected %s, got code=%d stderr=%q", labelPerm, code, stderr)
	}
}

func TestEdit_RefusesBadValues(t *testing.T) {
	for _, tc := range []struct{ name, value, want string }{
		{"empty", "", labelValueEmpty},
		{"embedded_newline", "aaaa\nbbbb", labelValueNL},
		{"unresolved_reference", "{{resolve:secretsmanager:x:SecretString:y}}", labelValueBad},
		{"not_hex64", "short", labelValueBad},
		{"nul_byte", "aa\x00bb", labelNUL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const before = "JWT_SECRET=old\n"
			path := writeEnv(t, before)
			code, _, stderr := runEditor(t, tc.value, "-file", path, "-expect-hex64")
			if code == 0 || strings.TrimSpace(stderr) != tc.want {
				t.Fatalf("expected %s, got code=%d stderr=%q", tc.want, code, stderr)
			}
			got, _ := os.ReadFile(path)
			if string(got) != before {
				t.Fatalf("rejected value must leave the file untouched: %q", got)
			}
		})
	}
}

func TestEdit_RefusesNulByteInFile(t *testing.T) {
	path := writeEnv(t, "JWT_SECRET=old\nBIN=\x00\n")
	code, _, stderr := runEditor(t, newVal, "-file", path)
	if code == 0 || strings.TrimSpace(stderr) != labelNUL {
		t.Fatalf("expected %s, got code=%d stderr=%q", labelNUL, code, stderr)
	}
}

func TestEdit_LeavesNoTemporaryFileBehind(t *testing.T) {
	path := writeEnv(t, "JWT_SECRET=old\nOTHER=x\n")
	dir := filepath.Dir(path)

	if code, _, stderr := runEditor(t, newVal, "-file", path); code != 0 {
		t.Fatalf("expected success, got %d %s", code, stderr)
	}
	// And after a refusal.
	if code, _, _ := runEditor(t, "", "-file", path); code == 0 {
		t.Fatal("empty value should have been refused")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".envsecret-") {
			t.Fatalf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestEdit_PreservesFileModeAndInodeReplacement(t *testing.T) {
	path := writeEnv(t, "JWT_SECRET=old\n")
	if code, _, stderr := runEditor(t, newVal, "-file", path); code != 0 {
		t.Fatalf("expected success, got %d %s", code, stderr)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode not preserved: %v", info.Mode().Perm())
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("target must remain a regular file")
	}
}

func TestRun_RejectsRelativePathAndBadKey(t *testing.T) {
	for _, args := range [][]string{
		{"-file", "relative/dev.env"},
		{"-file", "/tmp/x", "-key", "BAD=KEY"},
		{},
	} {
		code, _, stderr := runEditor(t, newVal, args...)
		if code != 2 || strings.TrimSpace(stderr) != labelUsage {
			t.Fatalf("args %v: expected %s exit 2, got code=%d stderr=%q",
				args, labelUsage, code, stderr)
		}
	}
}
