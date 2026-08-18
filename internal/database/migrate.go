package database

import (
	"fmt"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

// AutoMigrate applies GORM schema updates, then adds partial unique indexes
// so soft-deleted rows do not block reuse of email/slug.
//
// Chosen approach for the MVP: AutoMigrate on boot (solo + AI-friendly).
// SQL snapshots live in migrations/ for reference. A dedicated migrator
// can replace this later without changing models.
func AutoMigrate(db *gorm.DB, log zerolog.Logger) error {
	if db == nil {
		return nil
	}

	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.APIKey{},
		&models.RefreshToken{},
		&models.Contact{},
		&models.Conversation{},
		&models.Message{},
		&models.ProviderLog{},
		&models.Campaign{},
		&models.CampaignRecipient{},
		&models.TenantSetting{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}

	extras := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_slug_active ON tenants (slug) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email_active ON users (tenant_id, email) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_tenant_status_created ON conversations (tenant_id, status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_tenant_last_message ON conversations (tenant_id, last_message_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_tenant_conversation_created ON messages (tenant_id, conversation_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_tenant_phone ON contacts (tenant_id, phone)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_tenant_status ON campaigns (tenant_id, status, created_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_tenant_provider_msgid ON messages (tenant_id, provider, provider_message_id) WHERE provider_message_id <> '' AND deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_messages_tenant_created ON messages (tenant_id, created_at DESC)`,
	}
	for _, stmt := range extras {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("migrate index: %w", err)
		}
	}

	log.Info().Msg("database schema migrated")
	return nil
}
