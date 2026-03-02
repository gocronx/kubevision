package repository

import (
	"context"
	"testing"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestPluginDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.PluginConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestPluginConfigRepo_CRUD(t *testing.T) {
	db := setupTestPluginDB(t)
	repo := NewPluginConfigRepo(db)
	ctx := context.Background()

	// Create.
	pc := &model.PluginConfig{
		Name:       "prometheus",
		PluginType: "monitoring",
		Enabled:    true,
		Config:     `{"url":"http://prometheus:9090"}`,
	}
	if err := repo.Create(ctx, pc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pc.ID == 0 {
		t.Error("expected ID to be set after create")
	}

	// GetByName.
	got, err := repo.GetByName(ctx, "prometheus")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Name != "prometheus" {
		t.Errorf("expected name prometheus, got %s", got.Name)
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}

	// Update.
	got.Enabled = false
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, _ := repo.GetByName(ctx, "prometheus")
	if updated.Enabled {
		t.Error("expected enabled=false after update")
	}

	// List.
	pc2 := &model.PluginConfig{Name: "grafana", PluginType: "dashboard", Config: "{}"}
	_ = repo.Create(ctx, pc2)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(list))
	}

	// Delete.
	if err := repo.Delete(ctx, pc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list2, _ := repo.List(ctx)
	if len(list2) != 1 {
		t.Errorf("expected 1 plugin after delete, got %d", len(list2))
	}
}

func TestPluginConfigRepo_GetByName_NotFound(t *testing.T) {
	db := setupTestPluginDB(t)
	repo := NewPluginConfigRepo(db)

	_, err := repo.GetByName(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}
