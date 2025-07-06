package migrations

import (
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// Migrate performs the database migration for the application.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.McpServer{}); err != nil {
		return fmt.Errorf("auto‑migration failed for McpServer model: %v", err)
	}
	if err := db.AutoMigrate(&model.Tool{}); err != nil {
		return fmt.Errorf("auto‑migration failed for Tool model: %v", err)
	}
	if err := db.AutoMigrate(&model.ServerConfig{}); err != nil {
		return fmt.Errorf("auto‑migration failed for ServerConfig model: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		return fmt.Errorf("auto‑migration failed for User model: %v", err)
	}
	return nil
}
