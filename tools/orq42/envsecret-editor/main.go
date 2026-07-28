// Command envsecret-editor replaces the value of exactly one dotenv assignment
// in place, byte-preserving everything else, without ever emitting the value.
//
// ORQ-42 Q-H. It exists because the rotation runbook must not edit the durable
// dev.env with an ad-hoc shell one-liner: a `grep -v` rewrite masks read
// errors, recognises a single spelling, and can leave a file that contains only
// the new secret. This editor fails closed instead.
//
// Contract:
//
//	stdin   the new value, raw bytes, no trailing newline required
//	-file   absolute path of the dotenv file to edit
//	-key    assignment name (default JWT_SECRET)
//	-expect-hex64 require the new value to be exactly 64 hex characters
//	stdout  exactly one fixed label, never the value
//	exit    0 only on OK_EDIT
//
// Guarantees:
//
//	O_NOFOLLOW on the target, regular file, owner and mode checked
//	exactly one active assignment required; zero or many is a stop
//	prefix and suffix bytes are copied verbatim, so comments, ordering,
//	CRLF/LF and the terminal newline survive by construction
//	temp file created O_EXCL in the same directory with the same mode,
//	fsync(file) -> rename(2) -> fsync(dir)
//	temp removed on every exit path
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// Fixed labels. Nothing else is ever printed to stdout.
const (
	labelOK          = "OK_EDIT"
	labelPreflight   = "OK_PREFLIGHT"
	labelUsage       = "STOP_USAGE"
	labelKey         = "STOP_KEY"
	labelSymlink     = "STOP_SYMLINK"
	labelNotRegular  = "STOP_NOT_REGULAR"
	labelPerm        = "STOP_PERM"
	labelOwner       = "STOP_OWNER"
	labelRead        = "STOP_READ"
	labelNUL         = "STOP_NUL"
	labelNone        = "STOP_NONE"
	labelMulti       = "STOP_MULTI"
	labelVariant     = "STOP_VARIANT"
	labelNotCanon    = "STOP_NOT_CANONICAL"
	labelValueEmpty  = "STOP_VALUE_EMPTY"
	labelValueBad    = "STOP_VALUE_SHAPE"
	labelValueNL     = "STOP_VALUE_NEWLINE"
	labelWrite       = "STOP_WRITE"
	labelRename      = "STOP_RENAME"
	labelVerify      = "STOP_VERIFY"
	labelUnsupported = "STOP_UNSUPPORTED_PLATFORM"
)

// maxFileBytes caps the dotenv file. A dotenv large enough to exceed this is
// not the file this tool is meant to edit.
const maxFileBytes = 1 << 20

// maxValueBytes caps stdin. The rotation contract is a 64-character hex value;
// the cap only stops an unbounded read.
const maxValueBytes = 4096

var (
	hex64 = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	// dotenvKey is the accepted assignment name: a POSIX-style identifier. A key
	// carrying whitespace, a dot, a dash or a quote is refused outright rather
	// than matched loosely against file content.
	dotenvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type stopError struct{ label string }

func (e *stopError) Error() string { return e.label }

func stop(label string) error { return &stopError{label: label} }

// lineIterator calls fn for every line, giving the byte offsets of the line
// itself so callers can slice prefix and suffix without copying.
func lineIterator(data []byte, fn func(line []byte, start, end int)) {
	offset := 0
	for {
		lineEnd := bytes.IndexByte(data[offset:], '\n')
		if lineEnd < 0 {
			fn(data[offset:], offset, len(data))
			return
		}
		end := offset + lineEnd
		fn(data[offset:end], offset, end)
		offset = end + 1
		if offset > len(data) {
			return
		}
	}
}

// isSemanticVariant reports whether a line refers to the key in a spelling this
// tool refuses to touch: indented, `export`-prefixed, or with whitespace around
// the `=`. Those forms are legitimate dotenv in other tooling, which is exactly
// why they must stop the run instead of being edited or silently ignored.
func isSemanticVariant(line []byte, key string) bool {
	s := string(line)
	trimmed := strings.TrimLeft(s, " \t")
	indented := trimmed != s
	if strings.HasPrefix(trimmed, "#") {
		return false // a comment refers to nothing
	}
	body := trimmed
	exported := false
	if rest, ok := cutPrefixFold(body, "export"); ok && (strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\t")) {
		body = strings.TrimLeft(rest, " \t")
		exported = true
	}
	if !strings.HasPrefix(body, key) {
		return false
	}
	after := body[len(key):]
	spaced := strings.HasPrefix(after, " ") || strings.HasPrefix(after, "\t")
	afterTrim := strings.TrimLeft(after, " \t")
	if !strings.HasPrefix(afterTrim, "=") {
		return false // e.g. JWT_SECRETX=... is a different key
	}
	return indented || exported || spaced
}

// isCanonical reports whether the line is exactly `KEY=<64hex>`: no quoting, no
// inline comment, no surrounding whitespace.
func isCanonical(line []byte, key string) bool {
	prefix := key + "="
	if !bytes.HasPrefix(line, []byte(prefix)) {
		return false
	}
	return hex64.Match(line[len(prefix):])
}

func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return "", false
}

// assignmentSpan locates the single active `key=` assignment and returns the
// byte offsets of the value. It refuses zero, duplicates, and any semantic
// variant elsewhere in the file, so a mixed file (one canonical line plus an
// `export` line) stops instead of being half-edited.
func assignmentSpan(data []byte, key string) (valueStart, valueEnd int, err error) {
	found := 0
	variants := 0
	prefix := []byte(key + "=")
	lineIterator(data, func(line []byte, start, end int) {
		if bytes.HasPrefix(line, prefix) {
			found++
			if found == 1 {
				valueStart = start + len(prefix)
				valueEnd = end
				// Keep a trailing CR with the suffix so CRLF files survive.
				if valueEnd > valueStart && data[valueEnd-1] == '\r' {
					valueEnd--
				}
			}
			return
		}
		if isSemanticVariant(line, key) {
			variants++
		}
	})
	if variants > 0 {
		return 0, 0, stop(labelVariant)
	}
	switch found {
	case 0:
		return 0, 0, stop(labelNone)
	case 1:
		return valueStart, valueEnd, nil
	default:
		return 0, 0, stop(labelMulti)
	}
}

// openNoFollow opens path without following a final symlink and refuses
// anything that is not a regular file owned by the current user.
func openNoFollow(path string) (*os.File, os.FileInfo, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, nil, stop(labelSymlink)
		}
		return nil, nil, stop(labelRead)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, stop(labelRead)
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, nil, stop(labelNotRegular)
	}
	if info.Mode().Perm()&0o077 != 0 {
		f.Close()
		return nil, nil, stop(labelPerm)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		f.Close()
		return nil, nil, stop(labelUnsupported)
	}
	if int(st.Uid) != os.Getuid() {
		f.Close()
		return nil, nil, stop(labelOwner)
	}
	return f, info, nil
}

func readValue(r io.Reader, expectHex64 bool) ([]byte, error) {
	value, err := io.ReadAll(io.LimitReader(r, maxValueBytes+1))
	if err != nil {
		return nil, stop(labelRead)
	}
	if len(value) > maxValueBytes {
		return nil, stop(labelValueBad)
	}
	value = bytes.TrimRight(value, "\r\n")
	if len(value) == 0 {
		return nil, stop(labelValueEmpty)
	}
	if bytes.ContainsAny(value, "\r\n") {
		return nil, stop(labelValueNL)
	}
	if bytes.IndexByte(value, 0) >= 0 {
		return nil, stop(labelNUL)
	}
	if strings.Contains(string(value), "{{resolve:") {
		return nil, stop(labelValueBad)
	}
	if expectHex64 && !hex64.Match(value) {
		return nil, stop(labelValueBad)
	}
	return value, nil
}

// edit performs the replacement. It returns a fixed label error on every
// refusal and never includes file content or the value in any message.
func edit(path, key string, expectHex64 bool, stdin io.Reader) (retErr error) {
	value, err := readValue(stdin, expectHex64)
	if err != nil {
		return err
	}
	defer func() {
		for i := range value {
			value[i] = 0
		}
	}()

	f, info, err := openNoFollow(path)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return stop(labelRead)
	}
	if len(data) > maxFileBytes {
		return stop(labelRead)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return stop(labelNUL)
	}

	valueStart, valueEnd, err := assignmentSpan(data, key)
	if err != nil {
		return err
	}

	// Prefix and suffix are copied verbatim: comments, ordering, line endings
	// and the terminal newline are preserved because they are never rewritten.
	out := make([]byte, 0, len(data)-(valueEnd-valueStart)+len(value))
	out = append(out, data[:valueStart]...)
	out = append(out, value...)
	out = append(out, data[valueEnd:]...)

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".envsecret-*.tmp") // CreateTemp uses O_EXCL
	if err != nil {
		return stop(labelWrite)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return stop(labelWrite)
	}
	if _, err := tmp.Write(out); err != nil {
		return stop(labelWrite)
	}
	if err := tmp.Sync(); err != nil {
		return stop(labelWrite)
	}
	if err := tmp.Close(); err != nil {
		return stop(labelWrite)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return stop(labelRename)
	}
	committed = true

	// fsync the directory: rename(2) gives atomic visibility, not durability.
	d, err := os.Open(dir)
	if err != nil {
		return stop(labelRename)
	}
	if err := d.Sync(); err != nil {
		d.Close()
		return stop(labelRename)
	}
	if err := d.Close(); err != nil {
		return stop(labelRename)
	}

	// Verify by shape only: one assignment, and the non-target bytes unchanged.
	check, _, err := openNoFollow(path)
	if err != nil {
		return stop(labelVerify)
	}
	after, err := io.ReadAll(io.LimitReader(check, maxFileBytes+1))
	check.Close()
	if err != nil {
		return stop(labelVerify)
	}
	newStart, newEnd, err := assignmentSpan(after, key)
	if err != nil {
		return stop(labelVerify)
	}
	if !bytes.Equal(after[:newStart], data[:valueStart]) ||
		!bytes.Equal(after[newEnd:], data[valueEnd:]) {
		return stop(labelVerify)
	}
	return nil
}

// preflight opens the file, applies the same structural gates as edit, and
// additionally requires the single assignment to be exactly canonical. It reads
// stdin not at all and reports nothing about content beyond a fixed label, so it
// is safe to run before a rotation window.
func preflight(path, key string) error {
	f, _, err := openNoFollow(path)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	closeErr := f.Close()
	if err != nil || closeErr != nil || len(data) > maxFileBytes {
		return stop(labelRead)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return stop(labelNUL)
	}
	valueStart, _, err := assignmentSpan(data, key)
	if err != nil {
		return err
	}
	// Recover the whole line to assert the canonical shape.
	lineStart := bytes.LastIndexByte(data[:valueStart], '\n') + 1
	lineEnd := lineStart + bytes.IndexByte(data[lineStart:], '\n')
	if lineEnd < lineStart {
		lineEnd = len(data)
	}
	line := bytes.TrimRight(data[lineStart:lineEnd], "\r")
	if !isCanonical(line, key) {
		return stop(labelNotCanon)
	}
	return nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("envsecret-editor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "absolute path of the dotenv file")
	key := fs.String("key", "JWT_SECRET", "assignment name to replace")
	expectHex64 := fs.Bool("expect-hex64", false, "require a 64-character hex value")
	checkOnly := fs.Bool("check-only", false, "content-free preflight; never writes and never reads stdin")
	if err := fs.Parse(args); err != nil || *file == "" || !filepath.IsAbs(*file) {
		fmt.Fprintln(stderr, labelUsage)
		return 2
	}
	if !dotenvKey.MatchString(*key) {
		fmt.Fprintln(stderr, labelKey)
		return 2
	}

	var err error
	if *checkOnly {
		err = preflight(*file, *key)
	} else {
		err = edit(*file, *key, *expectHex64, stdin)
	}
	if err != nil {
		var s *stopError
		if errors.As(err, &s) {
			fmt.Fprintln(stderr, s.label)
		} else {
			fmt.Fprintln(stderr, labelWrite)
		}
		return 1
	}
	if *checkOnly {
		fmt.Fprintln(stdout, labelPreflight)
	} else {
		fmt.Fprintln(stdout, labelOK)
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
