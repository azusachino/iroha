// Command iroha-admin runs operator-only maintenance against the Iroha
// database. It ships in the server image next to iroha-server.
//
//	iroha-admin intake-token issue [-name primary]
//	iroha-admin intake-token list
//	iroha-admin intake-token revoke <cred_id>
//	iroha-admin password reset
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/dbconnect"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
	"gorm.io/gorm"
)

const usage = `usage:
  iroha-admin intake-token issue [-name primary]   issue a token for a device; prints it once
  iroha-admin intake-token list                    list credentials (never tokens)
  iroha-admin intake-token revoke <cred_id>        revoke one credential immediately
  iroha-admin owner setup [-username owner]        create initial owner account (reads password from stdin)
  iroha-admin password reset                       break-glass: read a new owner password from stdin, revoke all sessions
`

var errUsage = errors.New("invalid arguments")

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, errUsage) {
			fmt.Fprint(os.Stderr, usage)
		} else {
			fmt.Fprintln(os.Stderr, "iroha-admin:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) < 2 || (args[0] != "intake-token" && args[0] != "password" && args[0] != "owner") {
		return errUsage
	}
	cfg, err := config.Load("iroha.toml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	db, err := dbconnect.Connect(cfg.Database.URL, &gorm.Config{}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	credentials := intakecredential.NewService(db)
	ctx := context.Background()

	if args[0] == "owner" {
		if args[1] != "setup" {
			return errUsage
		}
		flags := flag.NewFlagSet("setup", flag.ContinueOnError)
		username := flags.String("username", "owner", "owner username")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		return setupOwner(ctx, auth.NewService(db), *username, os.Stdin)
	}

	if args[0] == "password" {
		if args[1] != "reset" || len(args) != 2 {
			return errUsage
		}
		return resetPassword(ctx, auth.NewService(db), os.Stdin)
	}

	switch args[1] {
	case "issue":
		flags := flag.NewFlagSet("issue", flag.ContinueOnError)
		name := flags.String("name", "primary", "device name; becomes the source instance iphone-hae:<name>")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		row, token, err := credentials.Issue(ctx, *name)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "issued %s (%s); older credentials stay active until revoked\n", ids.Encode(ids.IntakeCredentialPrefix, row.ID), row.Name)
		_, err = fmt.Fprintln(stdout, token)
		return err
	case "list":
		rows, err := credentials.List(ctx)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "ID\tNAME\tCREATED\tLAST USED\tREVOKED"); err != nil {
			return err
		}
		for _, row := range rows {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", ids.Encode(ids.IntakeCredentialPrefix, row.ID), row.Name,
				row.CreatedAt.Format(time.RFC3339), formatTime(row.LastUsedAt), formatTime(row.RevokedAt)); err != nil {
				return err
			}
		}
		return w.Flush()
	case "revoke":
		if len(args) != 3 {
			return errUsage
		}
		id, err := ids.Decode(ids.IntakeCredentialPrefix, args[2])
		if err != nil {
			return err
		}
		if err := credentials.Revoke(ctx, id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "revoked %s\n", args[2])
	default:
		return errUsage
	}
	return nil
}

// resetPassword reads the new password from the first line of stdin, so it
// never appears in shell history or the process list.
func resetPassword(ctx context.Context, service *auth.Service, stdin io.Reader) error {
	fmt.Fprintln(os.Stderr, "new owner password (one line on stdin):")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	username, err := service.ResetPassword(ctx, strings.TrimRight(line, "\r\n"))
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "password reset for %s; all sessions revoked\n", username)
	return nil
}

// setupOwner reads the initial owner password from stdin and creates the owner account.
func setupOwner(ctx context.Context, service *auth.Service, username string, stdin io.Reader) error {
	fmt.Fprintf(os.Stderr, "initial password for owner %q (one line on stdin):\n", username)
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	password := strings.TrimRight(line, "\r\n")
	token, principal, err := service.Setup(ctx, username, password)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "owner %q created successfully (user_id=%s)\n", principal.Username, principal.UserID)
	_ = token
	return nil
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
}
