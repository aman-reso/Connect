package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"Connect/pkg/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(connStr string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	p := &PostgresDB{db: db}
	if err := p.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize postgres schema: %w", err)
	}

	p.seedDefaultModels()
	p.RecoverInterruptedCalls()
	go p.startEphemeralCleaner()

	log.Println("🐘 PostgreSQL Database successfully connected and schema initialized!")
	return p, nil
}

func (p *PostgresDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(64) PRIMARY KEY,
		phone VARCHAR(20) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		role VARCHAR(20) NOT NULL DEFAULT 'user',
		avatar_url TEXT,
		bio TEXT,
		voice_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 10.00,
		group_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
		chat_rate_per_msg NUMERIC(10, 2) NOT NULL DEFAULT 1.00,
		is_online BOOLEAN NOT NULL DEFAULT FALSE,
		is_busy BOOLEAN NOT NULL DEFAULT FALSE,
		active_token TEXT,
		device_id VARCHAR(100),
		active_room_id VARCHAR(64),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
	CREATE INDEX IF NOT EXISTS idx_users_active_token ON users(active_token);

	CREATE TABLE IF NOT EXISTS wallets (
		user_id VARCHAR(64) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		balance NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		bonus_given NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		total_spent NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		total_earned NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id VARCHAR(64) PRIMARY KEY,
		user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		amount NUMERIC(12, 2) NOT NULL,
		type VARCHAR(50) NOT NULL,
		description TEXT,
		call_id VARCHAR(64),
		room_id VARCHAR(64),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);

	CREATE TABLE IF NOT EXISTS call_records (
		id VARCHAR(64) PRIMARY KEY,
		caller_id VARCHAR(64) NOT NULL REFERENCES users(id),
		caller_name VARCHAR(100) NOT NULL,
		receiver_id VARCHAR(64) NOT NULL REFERENCES users(id),
		receiver_name VARCHAR(100) NOT NULL,
		call_type VARCHAR(20) NOT NULL DEFAULT 'voice',
		status VARCHAR(30) NOT NULL,
		rate_per_min NUMERIC(10, 2) NOT NULL,
		started_at TIMESTAMP WITH TIME ZONE,
		ended_at TIMESTAMP WITH TIME ZONE,
		last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		duration_seconds INT NOT NULL DEFAULT 0,
		total_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		end_reason TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_call_records_caller ON call_records(caller_id);
	CREATE INDEX IF NOT EXISTS idx_call_records_receiver ON call_records(receiver_id);

	CREATE TABLE IF NOT EXISTS group_rooms (
		id VARCHAR(64) PRIMARY KEY,
		host_id VARCHAR(64) NOT NULL REFERENCES users(id),
		host_name VARCHAR(100) NOT NULL,
		host_avatar TEXT,
		title VARCHAR(200) NOT NULL,
		rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
		max_participants INT NOT NULL DEFAULT 10,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS ephemeral_messages (
		id VARCHAR(64) PRIMARY KEY,
		sender_id VARCHAR(64) NOT NULL REFERENCES users(id),
		receiver_id VARCHAR(64),
		room_id VARCHAR(64),
		content TEXT NOT NULL,
		cost NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_ephemeral_expires ON ephemeral_messages(expires_at);
	`
	_, err := p.db.Exec(schema)
	return err
}

func (p *PostgresDB) seedDefaultModels() {
	seedModels := []*models.User{
		{
			ID:              "model-1",
			Phone:           "9876543210",
			Name:            "Aanya Sharma",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=400&auto=format&fit=crop&q=80",
			Bio:             "Love deep late-night conversations, music & psychology 🌙",
			VoiceRatePerMin: 10.0,
			GroupRatePerMin: 5.0,
			ChatRatePerMsg:  1.0,
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
		{
			ID:              "model-2",
			Phone:           "9876543211",
			Name:            "Riya Sen",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=400&auto=format&fit=crop&q=80",
			Bio:             "Artist & traveler. Let's talk about dreams & coffee ☕✨",
			VoiceRatePerMin: 15.0,
			GroupRatePerMin: 7.0,
			ChatRatePerMsg:  2.0,
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
		{
			ID:              "model-3",
			Phone:           "9876543212",
			Name:            "Pooja Verma",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1524504388940-b1c1722653e1?w=400&auto=format&fit=crop&q=80",
			Bio:             "Friendly listener & anime enthusiast. Always here to cheer you up!",
			VoiceRatePerMin: 20.0,
			GroupRatePerMin: 8.0,
			ChatRatePerMsg:  2.5,
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
	}

	for _, m := range seedModels {
		token := fmt.Sprintf("token_%s_%s", m.ID, "seed")
		_, _ = p.db.Exec(`
			INSERT INTO users (id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (phone) DO UPDATE SET active_token = EXCLUDED.active_token
		`, m.ID, m.Phone, m.Name, m.Role, m.AvatarURL, m.Bio, m.VoiceRatePerMin, m.GroupRatePerMin, m.ChatRatePerMsg, true, false, token)

		_, _ = p.db.Exec(`
			INSERT INTO wallets (user_id, balance, bonus_given, total_spent, total_earned)
			VALUES ($1, 0, 0, 0, 0)
			ON CONFLICT (user_id) DO NOTHING
		`, m.ID)
	}
}

// 1. User & Single-Device Token Operations
func (p *PostgresDB) CreateOrLoginUser(phone, name string, role models.UserRole) (*models.User, string, bool, error) {
	// Check existing user
	var user models.User
	err := p.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, created_at
		FROM users WHERE phone = $1
	`, phone).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.VoiceRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.CreatedAt,
	)

	if err == nil {
		// Existing user: Generate new active token (invalidates old device)
		newToken := fmt.Sprintf("token_%s_%s", user.ID, uuid.New().String()[:8])
		_, err = p.db.Exec(`UPDATE users SET active_token = $1, is_online = TRUE WHERE id = $2`, newToken, user.ID)
		if err != nil {
			return nil, "", false, err
		}
		user.ActiveToken = newToken
		return &user, newToken, false, nil
	}

	// New user registration
	id := "user-" + uuid.New().String()[:8]
	if role == models.RoleModel {
		id = "model-" + uuid.New().String()[:8]
	}

	avatar := "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=400"
	if role == models.RoleModel {
		avatar = "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=400"
	}

	token := fmt.Sprintf("token_%s_%s", id, uuid.New().String()[:8])
	newUser := &models.User{
		ID:              id,
		Phone:           phone,
		Name:            name,
		Role:            role,
		AvatarURL:       avatar,
		Bio:             "Hey there! Connecting on the app.",
		VoiceRatePerMin: 12.0,
		GroupRatePerMin: 6.0,
		ChatRatePerMsg:  1.5,
		IsOnline:        true,
		IsBusy:          false,
		ActiveToken:     token,
		CreatedAt:       time.Now(),
	}

	tx, err := p.db.Begin()
	if err != nil {
		return nil, "", false, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO users (id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, newUser.ID, newUser.Phone, newUser.Name, newUser.Role, newUser.AvatarURL, newUser.Bio,
		newUser.VoiceRatePerMin, newUser.GroupRatePerMin, newUser.ChatRatePerMsg, true, false, token, newUser.CreatedAt)
	if err != nil {
		return nil, "", false, err
	}

	// Signup Welcome Incentive
	bonus := 0.0
	if role == models.RoleUser {
		bonus = 50.0 // ₹50 Signup Bonus
	}

	_, err = tx.Exec(`
		INSERT INTO wallets (user_id, balance, bonus_given, total_spent, total_earned, updated_at)
		VALUES ($1, $2, $3, 0, 0, NOW())
	`, newUser.ID, bonus, bonus)
	if err != nil {
		return nil, "", false, err
	}

	if bonus > 0 {
		_, err = tx.Exec(`
			INSERT INTO transactions (id, user_id, amount, type, description, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, uuid.New().String(), newUser.ID, bonus, models.TxTypeWelcomeBonus, "Welcome Bonus Incentive credited: ₹50.00")
		if err != nil {
			return nil, "", false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, "", false, err
	}

	return newUser, token, true, nil
}

func (p *PostgresDB) GetUserByToken(token string) (*models.User, error) {
	var user models.User
	err := p.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token, created_at
		FROM users WHERE active_token = $1
	`, token).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.VoiceRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.ActiveToken, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token or session expired on this device")
	}
	return &user, nil
}

func (p *PostgresDB) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := p.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token, created_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.VoiceRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.ActiveToken, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}

func (p *PostgresDB) ListModels() ([]*models.User, error) {
	rows, err := p.db.Query(`
		SELECT id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, created_at
		FROM users WHERE role = 'model' ORDER BY is_online DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.Name, &u.Role, &u.AvatarURL, &u.Bio,
			&u.VoiceRatePerMin, &u.GroupRatePerMin, &u.ChatRatePerMsg, &u.IsOnline, &u.IsBusy, &u.CreatedAt,
		); err == nil {
			list = append(list, &u)
		}
	}
	return list, rows.Err()
}

func (p *PostgresDB) SetUserPresence(id string, isOnline, isBusy bool) {
	_, _ = p.db.Exec(`UPDATE users SET is_online = $1, is_busy = $2 WHERE id = $3`, isOnline, isBusy, id)
}

// 2. ACID Wallets & Financial Ledger
func (p *PostgresDB) GetWallet(userID string) (*models.Wallet, error) {
	var w models.Wallet
	err := p.db.QueryRow(`
		SELECT user_id, balance, bonus_given, total_spent, total_earned, updated_at
		FROM wallets WHERE user_id = $1
	`, userID).Scan(&w.UserID, &w.Balance, &w.BonusGiven, &w.TotalSpent, &w.TotalEarned, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (p *PostgresDB) GetTransactions(userID string) ([]*models.Transaction, error) {
	rows, err := p.db.Query(`
		SELECT id, user_id, amount, type, description, COALESCE(call_id, ''), COALESCE(room_id, ''), created_at
		FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Type, &tx.Description, &tx.CallID, &tx.RoomID, &tx.CreatedAt); err == nil {
			list = append(list, &tx)
		}
	}
	return list, rows.Err()
}

func (p *PostgresDB) RechargeWallet(userID string, amount float64) (*models.Wallet, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var w models.Wallet
	err = tx.QueryRow(`
		UPDATE wallets SET balance = balance + $1, updated_at = NOW()
		WHERE user_id = $2
		RETURNING user_id, balance, bonus_given, total_spent, total_earned, updated_at
	`, amount, userID).Scan(&w.UserID, &w.Balance, &w.BonusGiven, &w.TotalSpent, &w.TotalEarned, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New().String(), userID, amount, models.TxTypeRecharge, fmt.Sprintf("Wallet Recharge of ₹%.2f", amount))
	if err != nil {
		return nil, err
	}

	return &w, tx.Commit()
}

func (p *PostgresDB) ProcessCallFinancials(callerID, receiverID, callID string, durationSec int, ratePerMin float64, endReason string) (float64, error) {
	if durationSec <= 0 {
		return 0, nil
	}

	minutes := float64(durationSec) / 60.0
	cost := minutes * ratePerMin

	tx, err := p.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var currentBalance float64
	err = tx.QueryRow(`SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE`, callerID).Scan(&currentBalance)
	if err != nil {
		return 0, err
	}

	if cost > currentBalance {
		cost = currentBalance
	}

	// Deduct from caller
	_, err = tx.Exec(`
		UPDATE wallets SET balance = balance - $1, total_spent = total_spent + $1, updated_at = NOW()
		WHERE user_id = $2
	`, cost, callerID)
	if err != nil {
		return 0, err
	}

	// Credit 80% to model
	modelShare := cost * 0.8
	_, err = tx.Exec(`
		UPDATE wallets SET balance = balance + $1, total_earned = total_earned + $1, updated_at = NOW()
		WHERE user_id = $2
	`, modelShare, receiverID)
	if err != nil {
		return 0, err
	}

	// Record Ledger
	_, _ = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, call_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New().String(), callerID, -cost, models.TxTypeCallDebit, fmt.Sprintf("Voice Call (%ds @ ₹%.1f/min) - %s", durationSec, ratePerMin, endReason), callID)

	_, _ = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, call_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New().String(), receiverID, modelShare, models.TxTypeCallCredit, fmt.Sprintf("Call Earnings (%ds @ ₹%.1f/min)", durationSec, ratePerMin), callID)

	return cost, tx.Commit()
}

// 3. Call History (METADATA ONLY - ZERO AUDIO FILES)
func (p *PostgresDB) CreateCallRecord(r *models.CallRecord) error {
	_, err := p.db.Exec(`
		INSERT INTO call_records (id, caller_id, caller_name, receiver_id, receiver_name, call_type, status, rate_per_min, started_at, ended_at, duration_seconds, total_cost, end_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, r.ID, r.CallerID, r.CallerName, r.ReceiverID, r.ReceiverName, r.CallType, r.Status, r.RatePerMin, r.StartedAt, r.EndedAt, r.DurationSeconds, r.TotalCost, r.EndReason, r.CreatedAt)
	return err
}

func (p *PostgresDB) UpdateCallRecord(r *models.CallRecord) error {
	_, err := p.db.Exec(`
		UPDATE call_records
		SET status = $1, started_at = $2, ended_at = $3, duration_seconds = $4, total_cost = $5, end_reason = $6
		WHERE id = $7
	`, r.Status, r.StartedAt, r.EndedAt, r.DurationSeconds, r.TotalCost, r.EndReason, r.ID)
	return err
}

func (p *PostgresDB) GetUserCallHistory(userID string) ([]*models.CallRecord, error) {
	rows, err := p.db.Query(`
		SELECT id, caller_id, caller_name, receiver_id, receiver_name, call_type, status, rate_per_min, started_at, ended_at, duration_seconds, total_cost, COALESCE(end_reason, ''), created_at
		FROM call_records
		WHERE caller_id = $1 OR receiver_id = $1
		ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.CallRecord
	for rows.Next() {
		var r models.CallRecord
		if err := rows.Scan(
			&r.ID, &r.CallerID, &r.CallerName, &r.ReceiverID, &r.ReceiverName, &r.CallType,
			&r.Status, &r.RatePerMin, &r.StartedAt, &r.EndedAt, &r.DurationSeconds, &r.TotalCost, &r.EndReason, &r.CreatedAt,
		); err == nil {
			list = append(list, &r)
		}
	}
	return list, rows.Err()
}

func (p *PostgresDB) UpdateCallHeartbeat(callID string) error {
	_, err := p.db.Exec(`UPDATE call_records SET last_heartbeat = NOW() WHERE id = $1 AND status = 'active'`, callID)
	return err
}

// 4. Server Crash Recovery Routine with Heartbeat Precision
func (p *PostgresDB) RecoverInterruptedCalls() {
	rows, err := p.db.Query(`
		SELECT id, caller_id, receiver_id, rate_per_min, started_at, last_heartbeat
		FROM call_records
		WHERE status = 'active'
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, callerID, receiverID string
		var ratePerMin float64
		var startedAt, lastHeartbeat *time.Time

		if err := rows.Scan(&id, &callerID, &receiverID, &ratePerMin, &startedAt, &lastHeartbeat); err == nil {
			durationSec := 0
			endedAt := time.Now()

			// Use the last confirmed 10-second heartbeat to avoid overcharging if server was down for hours!
			if startedAt != nil {
				if lastHeartbeat != nil && lastHeartbeat.After(*startedAt) {
					durationSec = int(lastHeartbeat.Sub(*startedAt).Seconds())
					endedAt = *lastHeartbeat
				} else {
					durationSec = 0
					endedAt = *startedAt
				}
			}

			// Process final financials up to the exact cut/crash heartbeat
			cost, _ := p.ProcessCallFinancials(callerID, receiverID, id, durationSec, ratePerMin, "Call cut during server interruption")

			// Mark call completed and reset model busy status
			_, _ = p.db.Exec(`
				UPDATE call_records
				SET status = 'completed', ended_at = $1, duration_seconds = $2, total_cost = $3, end_reason = 'Call cut during server interruption'
				WHERE id = $4
			`, endedAt, durationSec, cost, id)

			_, _ = p.db.Exec(`UPDATE users SET is_busy = FALSE WHERE id = $1`, receiverID)
			count++
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("⚠️ Server Crash Recovery: Error during row iteration: %v", err)
	}

	if count > 0 {
		log.Printf("🔄 Server Crash Recovery: Reconciled %d calls using last confirmed heartbeats.", count)
	}
}

// 5. Ephemeral Message Cleanup
func (p *PostgresDB) startEphemeralCleaner() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		_, _ = p.db.Exec(`DELETE FROM ephemeral_messages WHERE expires_at <= NOW()`)
	}
}
