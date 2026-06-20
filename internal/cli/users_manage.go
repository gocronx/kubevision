package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// Reset2FA clears a user's two-factor authentication so they can log in with a
// password again. Usage: kubevision reset-2fa --username admin
func Reset2FA(args []string) error {
	configPath, username, err := parseUserCmd("reset-2fa", args)
	if err != nil {
		return err
	}
	repo, err := userRepo(configPath)
	if err != nil {
		return err
	}
	if err := clear2FA(context.Background(), repo, username); err != nil {
		return err
	}
	fmt.Printf("Two-factor authentication cleared for user %q.\n", username)
	return nil
}

// SetRole changes a user's role and invalidates their sessions so the new role
// takes effect on next login. Usage: kubevision set-role --username dev --role admin
func SetRole(args []string) error {
	fs := flag.NewFlagSet("set-role", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config YAML file")
	username := fs.String("username", "", "username (required)")
	role := fs.String("role", "", "new role: super-admin|admin|editor|viewer (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" || *role == "" {
		return fmt.Errorf("--username and --role are required")
	}
	if !validRoles[*role] {
		return fmt.Errorf("invalid role %q (allowed: super-admin, admin, editor, viewer)", *role)
	}
	repo, err := userRepo(*configPath)
	if err != nil {
		return err
	}
	if err := setUserRole(context.Background(), repo, *username, *role); err != nil {
		return err
	}
	fmt.Printf("Role of user %q set to %q.\n", *username, *role)
	return nil
}

// ActivateUser re-enables a disabled account.
func ActivateUser(args []string) error {
	configPath, username, err := parseUserCmd("activate-user", args)
	if err != nil {
		return err
	}
	repo, err := userRepo(configPath)
	if err != nil {
		return err
	}
	if err := setUserActive(context.Background(), repo, username, true); err != nil {
		return err
	}
	fmt.Printf("User %q activated.\n", username)
	return nil
}

// DeactivateUser disables an account and invalidates its sessions.
func DeactivateUser(args []string) error {
	configPath, username, err := parseUserCmd("deactivate-user", args)
	if err != nil {
		return err
	}
	repo, err := userRepo(configPath)
	if err != nil {
		return err
	}
	if err := setUserActive(context.Background(), repo, username, false); err != nil {
		return err
	}
	fmt.Printf("User %q deactivated.\n", username)
	return nil
}

// DeleteUser permanently removes a user. Requires --force or an interactive
// confirmation. Usage: kubevision delete-user --username dev [--force]
func DeleteUser(args []string) error {
	fs := flag.NewFlagSet("delete-user", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config YAML file")
	username := fs.String("username", "", "username (required)")
	force := fs.Bool("force", false, "skip the confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" {
		return fmt.Errorf("--username is required")
	}
	if !*force {
		answer, err := readSecret(fmt.Sprintf("Delete user %q? Type the username to confirm: ", *username))
		if err != nil {
			return err
		}
		if strings.TrimSpace(answer) != *username {
			return fmt.Errorf("confirmation did not match; aborted")
		}
	}
	repo, err := userRepo(*configPath)
	if err != nil {
		return err
	}
	if err := deleteUserByName(context.Background(), repo, *username); err != nil {
		return err
	}
	fmt.Printf("User %q deleted.\n", *username)
	return nil
}

// ---- testable core operations (repo-level) ----

func clear2FA(ctx context.Context, repo repository.UserRepo, username string) error {
	user, err := mustGetUser(ctx, repo, username)
	if err != nil {
		return err
	}
	user.TOTPEnabled = false
	user.TOTPSecretEnc = ""
	user.RecoveryCodesEnc = ""
	return repo.Update(ctx, user)
}

func setUserRole(ctx context.Context, repo repository.UserRepo, username, role string) error {
	user, err := mustGetUser(ctx, repo, username)
	if err != nil {
		return err
	}
	// Demoting the last super-admin would lock everyone out of admin functions.
	if user.Role == "super-admin" && role != "super-admin" {
		if err := ensureNotLastSuperAdmin(ctx, repo, user); err != nil {
			return err
		}
	}
	user.Role = role
	user.TokenVersion++ // force re-login so the new role takes effect
	return repo.Update(ctx, user)
}

func setUserActive(ctx context.Context, repo repository.UserRepo, username string, active bool) error {
	user, err := mustGetUser(ctx, repo, username)
	if err != nil {
		return err
	}
	if !active {
		if err := ensureNotLastSuperAdmin(ctx, repo, user); err != nil {
			return err
		}
		user.TokenVersion++ // kill existing sessions on deactivation
	}
	user.IsActive = active
	return repo.Update(ctx, user)
}

func deleteUserByName(ctx context.Context, repo repository.UserRepo, username string) error {
	user, err := mustGetUser(ctx, repo, username)
	if err != nil {
		return err
	}
	if err := ensureNotLastSuperAdmin(ctx, repo, user); err != nil {
		return err
	}
	return repo.Delete(ctx, user.ID)
}

// ---- helpers ----

// parseUserCmd parses the common --config/--username flags shared by several
// single-user commands.
func parseUserCmd(name string, args []string) (configPath, username string, err error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cfg := fs.String("config", "", "path to config YAML file")
	user := fs.String("username", "", "username (required)")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if *user == "" {
		return "", "", fmt.Errorf("--username is required")
	}
	return *cfg, *user, nil
}

// userRepo opens the database and returns a UserRepo.
func userRepo(configPath string) (repository.UserRepo, error) {
	db, err := openDB(configPath)
	if err != nil {
		return nil, err
	}
	return repository.NewUserRepo(db), nil
}

// mustGetUser fetches a user by name, returning a clear error when absent.
func mustGetUser(ctx context.Context, repo repository.UserRepo, username string) (*model.User, error) {
	user, err := repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("look up user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user %q not found", username)
	}
	return user, nil
}

// ensureNotLastSuperAdmin refuses operations that would remove the final active
// super-admin, preventing an administrative lockout.
func ensureNotLastSuperAdmin(ctx context.Context, repo repository.UserRepo, target *model.User) error {
	if target.Role != "super-admin" {
		return nil
	}
	users, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	active := 0
	for _, u := range users {
		if u.Role == "super-admin" && u.IsActive {
			active++
		}
	}
	if active <= 1 {
		return fmt.Errorf("refusing: %q is the last active super-admin", target.Username)
	}
	return nil
}
