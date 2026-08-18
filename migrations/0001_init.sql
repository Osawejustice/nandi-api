-- Canonical schema snapshot for the Nandi MVP.
-- Runtime still uses GORM AutoMigrate on boot (documented in internal/database/migrate.go).
-- This file is the human-readable source of truth for indexes and tenant isolation.

-- tenants, users, api_keys, refresh_tokens
-- contacts, conversations, messages
-- provider_logs, campaigns, campaign_recipients, tenant_settings

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_slug_active ON tenants (slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email_active ON users (tenant_id, email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_status_created ON conversations (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_last_message ON conversations (tenant_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_tenant_conversation_created ON messages (tenant_id, conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_tenant_created ON messages (tenant_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_tenant_provider_msgid ON messages (tenant_id, provider, provider_message_id) WHERE provider_message_id <> '' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contacts_tenant_phone ON contacts (tenant_id, phone);
CREATE INDEX IF NOT EXISTS idx_campaigns_tenant_status ON campaigns (tenant_id, status, created_at DESC);
