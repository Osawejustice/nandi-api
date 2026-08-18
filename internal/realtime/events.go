package realtime

const (
	EventNewMessage          = "new_message"
	EventConversationCreated = "conversation_created"
	EventConversationUpdated = "conversation_updated"
	EventAgentPresence       = "agent_presence"
	EventCampaignUpdated     = "campaign_updated"
)

// Event is the small payload pushed over WebSocket and Redis.
// Keep it lean — the frontend fetches full records over REST.
type Event struct {
	Event          string `json:"event"`
	TenantID       string `json:"tenant_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`
	Payload        any    `json:"payload,omitempty"`
}
