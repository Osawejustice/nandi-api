package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/Osawejustice/nandi-api/internal/models"
	"github.com/Osawejustice/nandi-api/internal/realtime"
	"github.com/Osawejustice/nandi-api/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub  *realtime.Hub
	auth *services.AuthService
	log  zerolog.Logger
}

func NewWSHandler(hub *realtime.Hub, auth *services.AuthService, log zerolog.Logger) *WSHandler {
	return &WSHandler{hub: hub, auth: auth, log: log}
}

// Connect upgrades to a tenant-scoped WebSocket.
//
//	@Summary		WebSocket (pass access_token query or Bearer)
//	@Tags			realtime
//	@Param			access_token	query	string	false	"JWT access token"
//	@Success		101
//	@Router			/api/v1/ws [get]
func (h *WSHandler) Connect(c *gin.Context) {
	token := c.Query("access_token")
	if token == "" {
		if header := c.GetHeader("Authorization"); len(header) > 7 {
			token = header[7:]
		}
	}
	if token == "" {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing access token")
		return
	}
	principal, err := h.auth.AuthenticateJWT(token)
	if err != nil || principal == nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid access token")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn().Err(err).Msg("ws upgrade failed")
		return
	}

	client := h.hub.Register(principal.TenantID, principal.UserID)
	h.hub.Publish(c.Request.Context(), realtime.Event{
		Event:    realtime.EventAgentPresence,
		TenantID: principal.TenantID.String(),
		Payload:  map[string]any{"user_id": principal.UserID, "status": models.AgentStatusOnline},
	})

	go client.WritePump(conn)
	client.ReadPump(conn)
}
