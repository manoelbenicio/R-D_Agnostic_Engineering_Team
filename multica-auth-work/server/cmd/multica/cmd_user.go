package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

// User namespace exists so the daemon-injected `## Requesting User` brief
// has a CLI surface a human can mirror without having to construct
// PATCH /api/me by hand. Profile and password management live here; future
// per-user knobs (e.g. preferred language) should land as further subcommands
// rather than expand the verb surface elsewhere.

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Work with your user account",
}

var userProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Get or update your personal profile",
	Long: "Manage the personal profile that agents see when they pick up a task " +
		"on your behalf. The description is injected into the agent brief under " +
		"`## Requesting User`, so use it to share role, stack, and collaboration " +
		"preferences.",
}

var userProfileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show your current user profile",
	RunE:  runUserProfileGet,
}

var userProfileUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update your user profile (currently: profile description)",
	Long: "Set the personal profile description that gets injected into agent " +
		"briefs as `## Requesting User`. Pass an empty value to clear it.\n\n" +
		"Pick the input mode that preserves your content:\n" +
		"  --description \"...\"          inline (decodes \\n / \\t escapes)\n" +
		"  --description-stdin           pipe a HEREDOC (preserves verbatim)\n" +
		"  --description-file <path>     read a UTF-8 file (Windows-safe)\n",
	RunE: runUserProfileUpdate,
}

const (
	minUserPasswordCharacters = 12
	maxUserPasswordBytes      = 72
)

var userPasswordCmd = &cobra.Command{
	Use:   "password",
	Short: "Manage your password",
}

var userPasswordUpdateCmd = newUserPasswordUpdateCmd()

func newUserPasswordUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Create or replace your password",
		Long: "Create or replace your password. By default, the CLI reads it from " +
			"the terminal with echo disabled. For automation, pass --password-stdin; " +
			"password values are never accepted as arguments or flags.",
		Args: cobra.NoArgs,
		RunE: runUserPasswordUpdate,
	}
	cmd.Flags().Bool("password-stdin", false, "Read the password from stdin (explicit automation mode; keeps it out of shell history and process arguments)")
	return cmd
}

func init() {
	userCmd.AddCommand(userProfileCmd)
	userCmd.AddCommand(userPasswordCmd)
	userProfileCmd.AddCommand(userProfileGetCmd)
	userProfileCmd.AddCommand(userProfileUpdateCmd)
	userPasswordCmd.AddCommand(userPasswordUpdateCmd)

	userProfileGetCmd.Flags().String("output", "table", "Output format: table or json")

	userProfileUpdateCmd.Flags().String("description", "", "New profile description (decodes \\n, \\r, \\t, \\\\; pipe via --description-stdin to preserve literal backslashes)")
	userProfileUpdateCmd.Flags().Bool("description-stdin", false, "Read description from stdin (preserves multi-line content verbatim)")
	userProfileUpdateCmd.Flags().String("description-file", "", "Read description from a UTF-8 file (preserves multi-line content verbatim; use this on Windows when stdin piping mangles non-ASCII bytes)")
	userProfileUpdateCmd.Flags().Bool("clear", false, "Clear the profile description (equivalent to --description \"\")")
	userProfileUpdateCmd.Flags().String("output", "table", "Output format: table or json")
}

func runUserPasswordUpdate(cmd *cobra.Command, _ []string) error {
	fromStdin, _ := cmd.Flags().GetBool("password-stdin")
	var (
		password string
		err      error
	)
	if fromStdin {
		password, err = readPasswordInput(cmd.InOrStdin())
	} else {
		password, err = readPasswordFromTerminal(os.Stdin, cmd.ErrOrStderr())
	}
	if err != nil {
		return err
	}
	if err := validateUserPassword(password); err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	if err := client.PutJSON(ctx, "/api/me/password", map[string]string{"new_password": password}, nil); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Password updated.")
	return nil
}

func readPasswordInput(in io.Reader) (string, error) {
	return readBoundedPassword(in, false)
}

func readPasswordLine(in io.Reader) (string, error) {
	return readBoundedPassword(in, true)
}

func readBoundedPassword(in io.Reader, stopAtNewline bool) (string, error) {
	// Keep memory bounded without leaving an overlong terminal line queued for
	// the user's shell. We continue consuming input after the storage cap and
	// reject only after the line/stream is drained.
	const storedLimit = maxUserPasswordBytes + 2 // room for a CRLF terminator
	data := make([]byte, 0, storedLimit)
	tooLong := false
	buffer := make([]byte, 256)
	done := false
	for !done {
		n, err := in.Read(buffer)
		for _, b := range buffer[:n] {
			if len(data) < storedLimit {
				data = append(data, b)
			} else {
				tooLong = true
			}
			if stopAtNewline && b == '\n' {
				done = true
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("read password: %w", err)
		}
	}
	password := trimPasswordLineEnding(string(data))
	if tooLong {
		return "", fmt.Errorf("password must not exceed %d bytes", maxUserPasswordBytes)
	}
	if err := validateUserPassword(password); err != nil {
		return "", err
	}
	return password, nil
}

func trimPasswordLineEnding(password string) string {
	switch {
	case strings.HasSuffix(password, "\r\n"):
		return strings.TrimSuffix(password, "\r\n")
	case strings.HasSuffix(password, "\n"):
		return strings.TrimSuffix(password, "\n")
	case strings.HasSuffix(password, "\r"):
		return strings.TrimSuffix(password, "\r")
	default:
		return password
	}
}

func validateUserPassword(password string) error {
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}
	if !utf8.ValidString(password) {
		return fmt.Errorf("password must be valid UTF-8")
	}
	if utf8.RuneCountInString(password) < minUserPasswordCharacters {
		return fmt.Errorf("password must contain at least %d characters", minUserPasswordCharacters)
	}
	if len(password) > maxUserPasswordBytes {
		return fmt.Errorf("password must not exceed %d bytes", maxUserPasswordBytes)
	}
	return nil
}

func runUserProfileGet(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	var me map[string]any
	if err := client.GetJSON(ctx, "/api/me", &me); err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, me)
	}

	printUserProfileTable(os.Stdout, me)
	return nil
}

func runUserProfileUpdate(cmd *cobra.Command, _ []string) error {
	// `--clear` is its own flag (not "pass an empty string") because cobra's
	// default value for a Changed("") flag would otherwise be ambiguous with
	// "user typed `--description ""`". Keep both forms supported — the inline
	// empty string is what someone scripting bash would reach for.
	clearFlag, _ := cmd.Flags().GetBool("clear")
	desc, hasDesc, err := resolveTextFlag(cmd, "description")
	if err != nil {
		return err
	}

	if clearFlag && hasDesc {
		return fmt.Errorf("--clear cannot be combined with --description / --description-stdin / --description-file")
	}
	if !clearFlag && !hasDesc && !cmd.Flags().Changed("description") {
		return fmt.Errorf("nothing to update; pass --description, --description-stdin, --description-file, or --clear")
	}

	if clearFlag {
		desc = ""
	}

	body := map[string]any{"profile_description": desc}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	var me map[string]any
	if err := client.PatchJSON(ctx, "/api/me", body, &me); err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, me)
	}

	printUserProfileTable(os.Stdout, me)
	return nil
}

func printUserProfileTable(out *os.File, me map[string]any) {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "ID\t%s\n", strVal(me, "id"))
	fmt.Fprintf(w, "NAME\t%s\n", strVal(me, "name"))
	fmt.Fprintf(w, "EMAIL\t%s\n", strVal(me, "email"))
	desc := strVal(me, "profile_description")
	if desc == "" {
		desc = "(not set)"
	}
	fmt.Fprintf(w, "PROFILE DESCRIPTION\t%s\n", desc)
}
