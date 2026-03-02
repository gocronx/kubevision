package service

import (
	"context"
	"fmt"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/model"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// validRoles is the set of roles that may be assigned to a user.
var validRoles = map[string]bool{
	"super-admin": true,
	"admin":       true,
	"editor":      true,
	"viewer":      true,
	"custom":      true,
}

// UserService handles business logic for user management.
type UserService struct {
	repo repository.UserRepo
}

// NewUserService creates a new UserService.
func NewUserService(repo repository.UserRepo) *UserService {
	return &UserService{repo: repo}
}

// UserDetail is the public projection of a user — it never exposes the password hash.
type UserDetail struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

// toUserDetail converts a model.User to a UserDetail, dropping sensitive fields.
func toUserDetail(u *model.User) *UserDetail {
	return &UserDetail{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ListUsers returns all users without password hashes.
func (s *UserService) ListUsers(ctx context.Context) ([]UserDetail, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list users")
	}

	infos := make([]UserDetail, len(users))
	for i := range users {
		infos[i] = *toUserDetail(&users[i])
	}
	return infos, nil
}

// GetUser returns a single user by ID.
func (s *UserService) GetUser(ctx context.Context, id uint) (*UserDetail, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "user not found")
	}
	return toUserDetail(user), nil
}

// CreateUser creates a new user with a hashed password.
// It validates that the username is unique and that the role is valid.
func (s *UserService) CreateUser(ctx context.Context, username, password, role string) (*UserDetail, error) {
	if username == "" || password == "" {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "username and password are required")
	}
	if !validRoles[role] {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid role: %s", role))
	}

	// Check for duplicate username.
	if existing, _ := s.repo.GetByUsername(ctx, username); existing != nil {
		return nil, bizerr.New(bizerr.CodeConflict, "username already exists")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to hash password")
	}

	user := &model.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to create user")
	}

	return toUserDetail(user), nil
}

// UpdateUser updates the role and/or active status of a user.
// A user cannot deactivate themselves, and the super-admin role cannot be changed.
func (s *UserService) UpdateUser(ctx context.Context, id, callerID uint, role string, isActive bool) (*UserDetail, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	// Prevent changing the super-admin's role.
	if user.Role == "super-admin" && role != "super-admin" {
		return nil, bizerr.New(bizerr.CodeForbidden, "cannot change the role of the super-admin")
	}

	// Prevent self-deactivation.
	if id == callerID && !isActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "cannot deactivate your own account")
	}

	if role != "" {
		if !validRoles[role] {
			return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid role: %s", role))
		}
		user.Role = role
	}
	user.IsActive = isActive

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to update user")
	}

	return toUserDetail(user), nil
}

// DeleteUser removes a user by ID.
// A user cannot delete themselves, and the last remaining admin cannot be deleted.
func (s *UserService) DeleteUser(ctx context.Context, id, callerID uint) error {
	if id == callerID {
		return bizerr.New(bizerr.CodeForbidden, "cannot delete your own account")
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	// Guard against deleting the last admin.
	if user.Role == "admin" || user.Role == "super-admin" {
		users, err := s.repo.List(ctx)
		if err != nil {
			return bizerr.New(bizerr.CodeInternal, "failed to verify admin count")
		}
		adminCount := 0
		for _, u := range users {
			if u.Role == "admin" || u.Role == "super-admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return bizerr.New(bizerr.CodeForbidden, "cannot delete the last admin account")
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to delete user")
	}
	return nil
}

// ResetPassword allows an admin to set a new password for any user and
// bumps the token version to invalidate existing sessions.
func (s *UserService) ResetPassword(ctx context.Context, id uint, newPassword string) error {
	if newPassword == "" {
		return bizerr.New(bizerr.CodeParamInvalid, "new password is required")
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to hash password")
	}

	user.PasswordHash = hash
	user.TokenVersion++

	if err := s.repo.Update(ctx, user); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to reset password")
	}
	return nil
}

// ChangePassword lets an authenticated user change their own password.
// It verifies the old password before applying the change, then bumps the
// token version so all existing sessions (except the current one — that is the
// caller's responsibility) are invalidated.
func (s *UserService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	if oldPassword == "" || newPassword == "" {
		return bizerr.New(bizerr.CodeParamInvalid, "old and new passwords are required")
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	if !auth.CheckPassword(oldPassword, user.PasswordHash) {
		return bizerr.New(bizerr.CodeUnauthorized, "old password is incorrect")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to hash password")
	}

	user.PasswordHash = hash
	user.TokenVersion++

	if err := s.repo.Update(ctx, user); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to change password")
	}
	return nil
}
