package repository

import (
	"context"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/gorm"
)

// webhookRepo is the GORM-backed implementation of WebhookRepo.
type webhookRepo struct {
	db *gorm.DB
}

// NewWebhookRepo returns a new WebhookRepo backed by the given GORM database.
func NewWebhookRepo(db *gorm.DB) WebhookRepo {
	return &webhookRepo{db: db}
}

func (r *webhookRepo) Create(ctx context.Context, webhook *model.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

func (r *webhookRepo) GetByID(ctx context.Context, id uint) (*model.Webhook, error) {
	var webhook model.Webhook
	if err := r.db.WithContext(ctx).First(&webhook, id).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (r *webhookRepo) Update(ctx context.Context, webhook *model.Webhook) error {
	return r.db.WithContext(ctx).Save(webhook).Error
}

func (r *webhookRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Webhook{}, id).Error
}

func (r *webhookRepo) List(ctx context.Context) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	if err := r.db.WithContext(ctx).Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

func (r *webhookRepo) ListActive(ctx context.Context) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}
