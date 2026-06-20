// Package cli implements administrative subcommands for the kubevision binary
// (resetting passwords, creating users, listing users) so operators can manage
// accounts without the HTTP API — useful for recovery and bootstrap.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/term"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/repository"
)

// CommandFunc handles a subcommand given its own argument slice.
type CommandFunc func(args []string) error

// Commands maps subcommand names to their handlers.
var Commands = map[string]CommandFunc{
	"reset-password":  ResetPassword,
	"create-user":     CreateUser,
	"list-users":      ListUsers,
	"reset-2fa":       Reset2FA,
	"set-role":        SetRole,
	"activate-user":   ActivateUser,
	"deactivate-user": DeactivateUser,
	"delete-user":     DeleteUser,
}

// IsCommand reports whether name is a known administrative subcommand.
func IsCommand(name string) bool {
	_, ok := Commands[name]
	return ok
}

// Usage prints the available subcommands to stderr.
func Usage() {
	fmt.Fprintln(os.Stderr, "Usage: kubevision <command> [flags]")
	fmt.Fprintln(os.Stderr, "\nAdministrative commands:")
	fmt.Fprintln(os.Stderr, "  reset-password    Reset a user's password")
	fmt.Fprintln(os.Stderr, "  reset-2fa         Clear a user's two-factor authentication")
	fmt.Fprintln(os.Stderr, "  create-user       Create a new user")
	fmt.Fprintln(os.Stderr, "  list-users        List all users")
	fmt.Fprintln(os.Stderr, "  set-role          Change a user's role")
	fmt.Fprintln(os.Stderr, "  activate-user     Enable a disabled account")
	fmt.Fprintln(os.Stderr, "  deactivate-user   Disable an account")
	fmt.Fprintln(os.Stderr, "  delete-user       Permanently delete a user")
	fmt.Fprintln(os.Stderr, "\nWith no command (or 'serve'), the HTTP server starts.")
	fmt.Fprintln(os.Stderr, "Run 'kubevision <command> -h' for command-specific flags.")
}

// openDB loads configuration and opens the database, the way the server does.
// A no-op logger keeps administrative commands quiet.
func openDB(configPath string) (*gorm.DB, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	db, err := repository.NewDB(cfg, zap.NewNop())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Silence GORM's query logging; a CLI should print only its own output.
	return db.Session(&gorm.Session{Logger: gormlogger.Discard}), nil
}

// readSecret reads a password from stdin without echoing when attached to a
// terminal, falling back to a plain line read when piped (for scripting).
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
