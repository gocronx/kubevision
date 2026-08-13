package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"golang.org/x/term"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// minPasswordLen is the minimum length enforced for CLI-set passwords.
const minPasswordLen = 6

// validRoles are the roles assignable from the CLI.
var validRoles = map[string]bool{
	"super-admin": true,
	"admin":       true,
	"editor":      true,
	"viewer":      true,
}

// ResetPassword resets an existing user's password and invalidates their
// active sessions. Usage: kubevision reset-password --username admin [--password ...]
func ResetPassword(args []string) error {
	fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config YAML file")
	username := fs.String("username", "", "username to reset (required)")
	password := fs.String("password", "", "new password (prompted if omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" {
		return fmt.Errorf("--username is required")
	}

	newPassword, err := resolvePassword(*password)
	if err != nil {
		return err
	}

	db, err := openDB(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	repo := repository.NewUserRepo(db)
	if err := resetUserPassword(context.Background(), repo, *username, newPassword); err != nil {
		return err
	}
	fmt.Printf("Password reset for user %q. Existing sessions were invalidated.\n", *username)
	return nil
}

// CreateUser creates a new local user. Usage:
// kubevision create-user --username dev --role editor [--email ..] [--password ..]
func CreateUser(args []string) error {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config YAML file")
	username := fs.String("username", "", "username (required)")
	role := fs.String("role", "viewer", "role: super-admin|admin|editor|viewer")
	email := fs.String("email", "", "email (optional)")
	password := fs.String("password", "", "password (prompted if omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" {
		return fmt.Errorf("--username is required")
	}
	if !validRoles[*role] {
		return fmt.Errorf("invalid role %q (allowed: super-admin, admin, editor, viewer)", *role)
	}

	newPassword, err := resolvePassword(*password)
	if err != nil {
		return err
	}

	db, err := openDB(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	repo := repository.NewUserRepo(db)
	if err := createUser(context.Background(), repo, *username, newPassword, *role, *email); err != nil {
		return err
	}
	fmt.Printf("Created user %q with role %q.\n", *username, *role)
	return nil
}

// ListUsers prints all users. Usage: kubevision list-users [--config ...]
func ListUsers(args []string) error {
	fs := flag.NewFlagSet("list-users", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config YAML file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	db, err := openDB(*configPath)
	if err != nil {
		return err
	}
	defer closeDB(db)
	repo := repository.NewUserRepo(db)
	users, err := repo.List(context.Background())
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tROLE\tACTIVE\tPROVIDER\tEMAIL")
	for _, u := range users {
		fmt.Fprintf(w, "%d\t%s\t%s\t%t\t%s\t%s\n", u.ID, u.Username, u.Role, u.IsActive, u.AuthProvider, u.Email)
	}
	return w.Flush()
}

// resolvePassword returns the provided password or prompts for one (twice, with
// confirmation) when it was not supplied on the command line.
func resolvePassword(provided string) (string, error) {
	if provided != "" {
		return provided, nil
	}
	// When stdin is piped (scripting), read a single line without confirmation.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return readSecret("")
	}
	pw, err := readSecret("New password: ")
	if err != nil {
		return "", err
	}
	confirm, err := readSecret("Confirm password: ")
	if err != nil {
		return "", err
	}
	if pw != confirm {
		return "", fmt.Errorf("passwords do not match")
	}
	return pw, nil
}

// resetUserPassword hashes and stores a new password for username, bumping the
// token version so any issued JWTs stop working.
func resetUserPassword(ctx context.Context, repo repository.UserRepo, username, password string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	user, err := repo.GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("look up user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user %q not found", username)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = hash
	user.TokenVersion++ // invalidate existing sessions
	if err := repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// createUser inserts a new local user after validating uniqueness.
func createUser(ctx context.Context, repo repository.UserRepo, username, password, role, email string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	if existing, _ := repo.GetByUsername(ctx, username); existing != nil {
		return fmt.Errorf("user %q already exists", username)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return repo.Create(ctx, &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		AuthProvider: "local",
		IsActive:     true,
	})
}
