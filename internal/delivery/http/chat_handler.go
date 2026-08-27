package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"Connect/internal/dto"
)

// HandleChatConversations lists active conversations for the authenticated user.
func (h *HTTPHandler) HandleChatConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := h.extractUserID(r)
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	conversations, err := h.chatUC.GetConversations(userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res := &dto.ConversationListResponse{
		Count:         len(conversations),
		Conversations: conversations,
		Notice:        "🔒 Ephemeral 24h: Messages automatically disappear after 24 hours.",
	}
	SendJSON(w, http.StatusOK, "Conversations retrieved successfully", res)
}

// HandleChatMessages returns active unexpired messages between caller and partner.
func (h *HTTPHandler) HandleChatMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := h.extractUserID(r)
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	partnerID := r.URL.Query().Get("partner_id")
	if partnerID == "" {
		SendError(w, http.StatusBadRequest, "partner_id is required")
		return
	}

	msgs, err := h.chatUC.GetMessages(userID, partnerID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res := &dto.EphemeralChatResponse{
		PartnerID: partnerID,
		Messages:  msgs,
		Notice:    "🔒 Ephemeral 24h: All messages in this session expire 24 hours after creation.",
	}
	SendJSON(w, http.StatusOK, "Messages retrieved successfully", res)
}

// HandleSendChatMessage saves and relays a chat message with 24-hour expiration.
func (h *HTTPHandler) HandleSendChatMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := h.extractUserID(r)
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	var req dto.SendChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ReceiverID == "" || req.Content == "" {
		SendError(w, http.StatusBadRequest, "receiver_id and content are required")
		return
	}

	msg, err := h.chatUC.SendMessage(userID, req.ReceiverID, req.Content)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Message sent successfully", msg)
}

func (h *HTTPHandler) extractUserID(r *http.Request) string {
	userID := r.URL.Query().Get("user_id")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if token != "" {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil {
			return u.ID
		}
	}
	return userID
}
