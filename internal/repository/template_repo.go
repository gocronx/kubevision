package repository

import (
	"context"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/gorm"
)

// templateRepo is the GORM-backed implementation of TemplateRepo.
type templateRepo struct {
	db *gorm.DB
}

// NewTemplateRepo returns a new TemplateRepo backed by the given GORM database.
func NewTemplateRepo(db *gorm.DB) TemplateRepo {
	return &templateRepo{db: db}
}

func (r *templateRepo) Create(ctx context.Context, tmpl *model.Template) error {
	return r.db.WithContext(ctx).Create(tmpl).Error
}

func (r *templateRepo) GetByID(ctx context.Context, id uint) (*model.Template, error) {
	var tmpl model.Template
	if err := r.db.WithContext(ctx).First(&tmpl, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tmpl, nil
}

func (r *templateRepo) Update(ctx context.Context, tmpl *model.Template) error {
	return r.db.WithContext(ctx).Save(tmpl).Error
}

func (r *templateRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Template{}, id).Error
}

func (r *templateRepo) List(ctx context.Context, category string) ([]model.Template, error) {
	var templates []model.Template
	q := r.db.WithContext(ctx).Order("is_builtin DESC, name ASC")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}
