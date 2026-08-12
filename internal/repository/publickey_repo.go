package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCeremonyUnavailable = errors.New("public key ceremony unavailable")

type PublicKeyRepo struct {
	db *gorm.DB
}

func NewPublicKeyRepo(db *gorm.DB) *PublicKeyRepo { return &PublicKeyRepo{db: db} }

func (r *PublicKeyRepo) SetUserHandle(ctx context.Context, userID uint, handle []byte) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND (public_key_handle IS NULL OR length(public_key_handle) = 0)", userID).
		Update("public_key_handle", handle)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *PublicKeyRepo) FindUserByHandle(ctx context.Context, handle []byte) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("public_key_handle = ?", handle).First(&user).Error
	return &user, err
}

func (r *PublicKeyRepo) CreateCeremony(ctx context.Context, ceremony *model.PublicKeyCeremony) error {
	return r.db.WithContext(ctx).Create(ceremony).Error
}

func (r *PublicKeyRepo) ConsumeCeremony(ctx context.Context, id, kind string, now time.Time) (*model.PublicKeyCeremony, error) {
	var ceremony model.PublicKeyCeremony
	result := r.db.WithContext(ctx).Model(&ceremony).Clauses(clause.Returning{}).
		Where("id = ? AND kind = ? AND consumed_at IS NULL AND expires_at > ?", id, kind, now).
		Update("consumed_at", now)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrCeremonyUnavailable
	}
	return &ceremony, nil
}

func (r *PublicKeyRepo) CreateCredential(ctx context.Context, credential *model.PublicKeyCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *PublicKeyRepo) ListActive(ctx context.Context, userID uint) ([]model.PublicKeyCredential, error) {
	var credentials []model.PublicKeyCredential
	err := r.db.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at ASC").Find(&credentials).Error
	return credentials, err
}

func (r *PublicKeyRepo) FindActiveByCredentialID(ctx context.Context, credentialID []byte) (*model.PublicKeyCredential, error) {
	var credential model.PublicKeyCredential
	err := r.db.WithContext(ctx).Where("credential_id = ? AND revoked_at IS NULL", credentialID).First(&credential).Error
	return &credential, err
}

func (r *PublicKeyRepo) Rename(ctx context.Context, userID, id uint, label string) error {
	result := r.db.WithContext(ctx).Model(&model.PublicKeyCredential{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).Update("label", label)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *PublicKeyRepo) Revoke(ctx context.Context, actorID, credentialID uint, userHasOtherMethod bool, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&owner, actorID).Error; err != nil {
			return err
		}
		var credential model.PublicKeyCredential
		if err := tx.Where("id = ? AND user_id = ? AND revoked_at IS NULL", credentialID, actorID).First(&credential).Error; err != nil {
			return err
		}
		if !userHasOtherMethod {
			var count int64
			if err := tx.Model(&model.PublicKeyCredential{}).
				Where("user_id = ? AND revoked_at IS NULL", actorID).Count(&count).Error; err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastAuthenticationMethod
			}
		}
		return tx.Model(&credential).Where("revoked_at IS NULL").Update("revoked_at", now).Error
	})
}

func (r *PublicKeyRepo) UpdateAfterLogin(ctx context.Context, id uint, previousCount, signCount uint32, backupState bool, usedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.PublicKeyCredential{}).
		Where("id = ? AND revoked_at IS NULL AND sign_count = ?", id, previousCount).
		Updates(map[string]any{"sign_count": signCount, "backup_state": backupState, "last_used_at": usedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCounterRace
	}
	return nil
}

var ErrLastAuthenticationMethod = errors.New("cannot remove the last authentication method")
var ErrCounterRace = errors.New("credential counter changed concurrently")
