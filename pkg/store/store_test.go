package store_test

import (
	"testing"
	"time"

	"Connect/pkg/models"
	"Connect/pkg/store"
)

func TestSignupBonusIncentive(t *testing.T) {
	st := store.NewStore()

	user, token, isNew := st.CreateOrLoginUser("9999999999", "Test User", models.RoleUser)
	if !isNew {
		t.Fatalf("Expected new user")
	}
	if token == "" {
		t.Fatalf("Expected valid token")
	}

	wallet, err := st.GetWallet(user.ID)
	if err != nil {
		t.Fatalf("Failed to fetch wallet: %v", err)
	}

	if wallet.Balance != store.WelcomeBonusAmount {
		t.Fatalf("Expected ₹%.2f bonus balance, got ₹%.2f", store.WelcomeBonusAmount, wallet.Balance)
	}
}

func TestCallFinancialsAndHistory(t *testing.T) {
	st := store.NewStore()

	caller, _, _ := st.CreateOrLoginUser("9999988888", "Rohan", models.RoleUser)
	modelList := st.ListModels()
	if len(modelList) == 0 {
		t.Fatalf("Expected default models to be seeded")
	}
	targetModel := modelList[0]

	callID := "test_call_001"
	record := &models.CallRecord{
		ID:           callID,
		CallerID:     caller.ID,
		CallerName:   caller.Name,
		ReceiverID:   targetModel.ID,
		ReceiverName: targetModel.Name,
		CallType:     "voice",
		Status:       models.CallStatusActive,
		RatePerMin:   targetModel.VoiceRatePerMin,
		CreatedAt:    time.Now(),
	}
	st.CreateCallRecord(record)

	cost, err := st.ProcessCallFinancials(caller.ID, targetModel.ID, callID, 120, targetModel.VoiceRatePerMin, "Call completed normally")
	if err != nil {
		t.Fatalf("ProcessCallFinancials failed: %v", err)
	}

	expectedCost := 2.0 * targetModel.VoiceRatePerMin
	if cost != expectedCost {
		t.Fatalf("Expected cost of ₹%.2f, got ₹%.2f", expectedCost, cost)
	}

	callerWallet, _ := st.GetWallet(caller.ID)
	expectedBalance := 50.0 - expectedCost
	if callerWallet.Balance != expectedBalance {
		t.Fatalf("Expected caller balance ₹%.2f, got ₹%.2f", expectedBalance, callerWallet.Balance)
	}

	history := st.GetUserCallHistory(caller.ID)
	if len(history) != 1 {
		t.Fatalf("Expected 1 call record in history, got %d", len(history))
	}
}

func TestGroupRoomMultiUserBilling(t *testing.T) {
	st := store.NewStore()

	// 1. Host Model creates group room
	host, _, _ := st.CreateOrLoginUser("9876543210", "Aanya Sharma", models.RoleModel)
	room, err := st.CreateGroupRoom(host, "Late Night Lounge", 6.0) // ₹6/min
	if err != nil {
		t.Fatalf("Failed to create group room: %v", err)
	}

	// 2. User 1 & User 2 join group room
	user1, _, _ := st.CreateOrLoginUser("9000000001", "User One", models.RoleUser) // ₹50 bonus
	user2, _, _ := st.CreateOrLoginUser("9000000002", "User Two", models.RoleUser) // ₹50 bonus

	_, err = st.JoinGroupRoom(room.ID, user1)
	if err != nil {
		t.Fatalf("User 1 failed to join: %v", err)
	}

	_, err = st.JoinGroupRoom(room.ID, user2)
	if err != nil {
		t.Fatalf("User 2 failed to join: %v", err)
	}

	// 3. User 1 leaves after 60s (1 min @ ₹6/min = ₹6.00)
	cost1, dur1, _ := st.LeaveGroupRoom(room.ID, user1.ID, "User left")
	if dur1 < 0 {
		t.Fatalf("Invalid duration")
	}

	w1, _ := st.GetWallet(user1.ID)
	if w1.Balance > 50.0 {
		t.Fatalf("Expected balance deduction for User 1")
	}

	// 4. Verify room is still active with User 2 and Host
	activeRoom, err := st.GetGroupRoom(room.ID)
	if err != nil || !activeRoom.IsActive {
		t.Fatalf("Room should still be active for other users")
	}
	if _, ok := activeRoom.Participants[user2.ID]; !ok {
		t.Fatalf("User 2 should still be in room")
	}

	// 5. User 2 leaves
	cost2, _, _ := st.LeaveGroupRoom(room.ID, user2.ID, "User 2 left")
	if cost2 < 0 {
		t.Fatalf("Invalid cost %v", cost1)
	}
}

func TestEphemeralChatExpiry(t *testing.T) {
	st := store.NewStore()

	u1, _, _ := st.CreateOrLoginUser("9111111111", "Sender", models.RoleUser)
	u2, _, _ := st.CreateOrLoginUser("9222222222", "Receiver", models.RoleUser)

	msg := &models.EphemeralMessage{
		ID:         "msg_001",
		SenderID:   u1.ID,
		ReceiverID: u2.ID,
		Content:    "Secret message",
		ExpiresAt:  time.Now().Add(100 * time.Millisecond),
		CreatedAt:  time.Now(),
	}
	st.SaveEphemeralMessage(msg)

	active := st.GetEphemeralMessages(u1.ID, u2.ID)
	if len(active) != 1 {
		t.Fatalf("Expected 1 active message, got %d", len(active))
	}

	time.Sleep(150 * time.Millisecond)
	expired := st.GetEphemeralMessages(u1.ID, u2.ID)
	if len(expired) != 0 {
		t.Fatalf("Expected 0 active messages after expiry, got %d", len(expired))
	}
}
