package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/repository"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db         *sql.DB
	Users      repository.UserRepository
	Wallets    repository.WalletRepository
	Calls      repository.CallRepository
	Rooms      repository.RoomRepository
	Messages   repository.MessageRepository
	Payments   repository.PaymentRepository
	Onboarding repository.ModelOnboardingRepository
	Reports    repository.ReportRepository
	Favorites  repository.FavoriteRepository
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres open error: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping error: %w", err)
	}

	store := &PostgresStore{
		db:         db,
		Users:      &userRepo{db: db},
		Wallets:    &walletRepo{db: db},
		Calls:      &callRepo{db: db, wallets: &walletRepo{db: db}},
		Rooms:      &roomRepo{db: db, wallets: &walletRepo{db: db}},
		Messages:   &messageRepo{db: db},
		Payments:   &paymentRepo{db: db},
		Onboarding: &modelOnboardingRepo{db: db},
		Reports:    &reportRepo{db: db},
		Favorites:  &favoriteRepo{db: db},
	}

	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("postgres schema error: %w", err)
	}

	store.seedDefaultModels()
	_ = store.Calls.RecoverInterruptedCalls()
	go store.startEphemeralCleaner()

	log.Println("🐘 PostgreSQL Clean Architecture Repository connected!")
	return store, nil
}

func (s *PostgresStore) initSchema() error {
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
		balance NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
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

	CREATE TABLE IF NOT EXISTS payment_orders (
		id VARCHAR(64) PRIMARY KEY,
		original_payment_id VARCHAR(64),
		user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		amount NUMERIC(12, 2) NOT NULL,
		currency VARCHAR(10) NOT NULL DEFAULT 'INR',
		status VARCHAR(30) NOT NULL,
		gateway_name VARCHAR(50) NOT NULL DEFAULT 'razorpay',
		gateway_order_id VARCHAR(100),
		gateway_payment_id VARCHAR(100),
		gateway_signature TEXT,
		failure_reason TEXT,
		refund_reason TEXT,
		refund_id VARCHAR(100),
		retry_count INT NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		completed_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_payment_orders_user ON payment_orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);
	CREATE INDEX IF NOT EXISTS idx_payment_orders_original ON payment_orders(original_payment_id);

	CREATE TABLE IF NOT EXISTS payment_audit_logs (
		id VARCHAR(64) PRIMARY KEY,
		payment_id VARCHAR(64) NOT NULL REFERENCES payment_orders(id) ON DELETE CASCADE,
		from_status VARCHAR(30) NOT NULL,
		to_status VARCHAR(30) NOT NULL,
		event_name VARCHAR(100) NOT NULL,
		gateway_ref_id VARCHAR(100),
		gateway_code VARCHAR(50),
		message TEXT NOT NULL,
		metadata_json TEXT,
		duration_ms BIGINT NOT NULL DEFAULT 0,
		started_at TIMESTAMP WITH TIME ZONE NOT NULL,
		ended_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_payment_audit_payment ON payment_audit_logs(payment_id);
	CREATE INDEX IF NOT EXISTS idx_payment_audit_created ON payment_audit_logs(created_at ASC);

	CREATE TABLE IF NOT EXISTS model_profiles (
		id VARCHAR(64) PRIMARY KEY,
		user_id VARCHAR(64) UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		display_name VARCHAR(100) NOT NULL,
		bio TEXT NOT NULL,
		avatar_url TEXT NOT NULL,
		age INT NOT NULL CHECK (age >= 18),
		gender VARCHAR(20) NOT NULL,
		languages VARCHAR(200) NOT NULL,
		interests VARCHAR(200) NOT NULL,
		voice_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 15.00,
		group_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 7.00,
		chat_rate_per_msg NUMERIC(10, 2) NOT NULL DEFAULT 2.00,
		payout_upi VARCHAR(100),
		payout_bank_acc VARCHAR(50),
		payout_ifsc VARCHAR(20),
		audio_intro_url TEXT,
		status VARCHAR(30) NOT NULL DEFAULT 'pending_review',
		rejection_reason TEXT,
		report_count INT NOT NULL DEFAULT 0,
		is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_model_profiles_user ON model_profiles(user_id);
	CREATE INDEX IF NOT EXISTS idx_model_profiles_status ON model_profiles(status);

	CREATE TABLE IF NOT EXISTS model_reports (
		id VARCHAR(64) PRIMARY KEY,
		reporter_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		reporter_name VARCHAR(100) NOT NULL,
		model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		model_name VARCHAR(100) NOT NULL,
		call_id VARCHAR(64),
		room_id VARCHAR(64),
		category VARCHAR(50) NOT NULL,
		description TEXT NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'submitted',
		admin_action VARCHAR(50) DEFAULT 'none',
		admin_note TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		resolved_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_model_reports_model ON model_reports(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_reports_reporter ON model_reports(reporter_id);
	CREATE INDEX IF NOT EXISTS idx_model_reports_status ON model_reports(status);

	CREATE TABLE IF NOT EXISTS user_favorites (
		user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		PRIMARY KEY (user_id, model_id)
	);

	CREATE INDEX IF NOT EXISTS idx_user_favorites_user ON user_favorites(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_favorites_model ON user_favorites(model_id);

	-- Schema Migrations for advanced discovery, location & full onboarding
	ALTER TABLE users ADD COLUMN IF NOT EXISTS age INT DEFAULT 21;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS gender VARCHAR(20) DEFAULT 'female';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS city VARCHAR(100);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS state VARCHAR(100);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS country VARCHAR(100) DEFAULT 'India';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS latitude NUMERIC(10, 6);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS longitude NUMERIC(10, 6);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS rating NUMERIC(3, 2) NOT NULL DEFAULT 4.90;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS review_count INT NOT NULL DEFAULT 0;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS total_calls_count INT NOT NULL DEFAULT 0;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS total_minutes_spoken INT NOT NULL DEFAULT 0;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS video_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 20.00;

	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS full_legal_name VARCHAR(150);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS gallery_urls TEXT;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS date_of_birth VARCHAR(30);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS govt_id_type VARCHAR(50);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS govt_id_number VARCHAR(100);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS govt_id_doc_url TEXT;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS selfie_verification_url TEXT;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS city VARCHAR(100);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS state VARCHAR(100);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS country VARCHAR(100) DEFAULT 'India';
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS pincode VARCHAR(20);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS address_line TEXT;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS latitude NUMERIC(10, 6);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS longitude NUMERIC(10, 6);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS video_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 20.00;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS payout_method VARCHAR(30) DEFAULT 'upi';
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS payout_beneficiary_name VARCHAR(150);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS pan_number VARCHAR(20);
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS agreed_to_safety_guidelines BOOLEAN NOT NULL DEFAULT TRUE;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS agreed_to_terms BOOLEAN NOT NULL DEFAULT TRUE;
	ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS safety_accepted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStore) seedDefaultModels() {
	// Clean State: No mock models. Pure real registered accounts.
}

func (s *PostgresStore) startEphemeralCleaner() {
	ticker := time.NewTicker(20 * time.Second)
	for range ticker.C {
		_ = s.Messages.PurgeExpired()
	}
}

// ----------------- USER REPO -----------------
type userRepo struct{ db *sql.DB }

func (r *userRepo) CreateOrLogin(phone, name string, role domain.UserRole) (*domain.User, string, bool, error) {
	var user domain.User
	err := r.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, voice_rate_per_min, group_rate_per_min, chat_rate_per_msg, is_online, is_busy, created_at
		FROM users WHERE phone = $1
	`, phone).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.VoiceRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.CreatedAt,
	)

	if err == nil {
		newToken := fmt.Sprintf("token_%s_%s", user.ID, uuid.New().String()[:8])
		if role != "" && user.Role != role {
			user.Role = role
			if name != "" {
				user.Name = name
			}
			_, _ = r.db.Exec(`UPDATE users SET active_token = $1, role = $2, name = $3, is_online = TRUE WHERE id = $4`, newToken, role, user.Name, user.ID)
		} else {
			_, _ = r.db.Exec(`UPDATE users SET active_token = $1, is_online = TRUE WHERE id = $2`, newToken, user.ID)
		}
		if role == domain.RoleModel {
			_, _ = r.db.Exec(`
				INSERT INTO model_profiles (id, user_id, display_name, bio, avatar_url, age, gender, city, state, country, latitude, longitude, languages, interests, voice_rate_per_min, video_rate_per_min, group_rate_per_min, chat_rate_per_msg, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, 21, 'female', 'New Delhi', 'Delhi', 'India', 28.6139, 77.2090, 'English, Hindi', 'Conversations, Music', $6, $7, $8, $9, 'approved', NOW(), NOW())
				ON CONFLICT (user_id) DO UPDATE SET status = 'approved', updated_at = NOW()
			`, "prof_"+user.ID, user.ID, user.Name, user.Bio, user.AvatarURL,
				user.VoiceRatePerMin, user.VoiceRatePerMin*1.5, user.GroupRatePerMin, user.ChatRatePerMsg)
		}
		user.ActiveToken = newToken
		return &user, newToken, false, nil
	}

	id := "user-" + uuid.New().String()[:8]
	if role == domain.RoleModel {
		id = "model-" + uuid.New().String()[:8]
	}

	avatar := "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=400"
	if role == domain.RoleModel {
		avatar = "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=400"
	}

	token := fmt.Sprintf("token_%s_%s", id, uuid.New().String()[:8])
	newUser := &domain.User{
		ID:              id,
		Phone:           phone,
		Name:            name,
		Role:            role,
		AvatarURL:       avatar,
		Bio:             "Hey there! Connecting on Connect.",
		VoiceRatePerMin: 12.0,
		GroupRatePerMin: 6.0,
		ChatRatePerMsg:  1.5,
		IsOnline:        true,
		IsBusy:          false,
		ActiveToken:     token,
		CreatedAt:       time.Now(),
	}

	tx, err := r.db.Begin()
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

	if role == domain.RoleModel {
		_, err = tx.Exec(`
			INSERT INTO model_profiles (id, user_id, display_name, bio, avatar_url, age, gender, city, state, country, latitude, longitude, languages, interests, voice_rate_per_min, video_rate_per_min, group_rate_per_min, chat_rate_per_msg, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 21, 'female', 'New Delhi', 'Delhi', 'India', 28.6139, 77.2090, 'English, Hindi', 'Conversations, Music', $6, $7, $8, $9, 'approved', NOW(), NOW())
			ON CONFLICT (user_id) DO UPDATE SET 
				display_name = EXCLUDED.display_name,
				bio = EXCLUDED.bio,
				avatar_url = EXCLUDED.avatar_url,
				status = 'approved',
				updated_at = NOW()
		`, "prof_"+newUser.ID, newUser.ID, newUser.Name, newUser.Bio, newUser.AvatarURL,
			newUser.VoiceRatePerMin, newUser.VoiceRatePerMin*1.5, newUser.GroupRatePerMin, newUser.ChatRatePerMsg)
		if err != nil {
			return nil, "", false, err
		}
	}

	bonus := 0.0
	if role == domain.RoleUser {
		bonus = 50.0
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
		`, uuid.New().String(), newUser.ID, bonus, domain.TxTypeWelcomeBonus, "Welcome Bonus Incentive credited: ₹50.00")
		if err != nil {
			return nil, "", false, err
		}
	}

	return newUser, token, true, tx.Commit()
}

func (r *userRepo) GetByToken(token string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, COALESCE(age, 21), COALESCE(gender, 'female'), COALESCE(city, ''), COALESCE(state, ''), COALESCE(country, 'India'), COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(rating, 4.90), COALESCE(review_count, 0), COALESCE(total_calls_count, 0), COALESCE(total_minutes_spoken, 0), voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token, created_at
		FROM users WHERE active_token = $1
	`, token).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.Age, &user.Gender, &user.City, &user.State, &user.Country, &user.Latitude, &user.Longitude,
		&user.Rating, &user.ReviewCount, &user.TotalCallsCount, &user.TotalMinutesSpoken,
		&user.VoiceRatePerMin, &user.VideoRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.ActiveToken, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token or session logged in on another device")
	}
	return &user, nil
}

func (r *userRepo) GetByID(id string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(`
		SELECT id, phone, name, role, avatar_url, bio, COALESCE(age, 21), COALESCE(gender, 'female'), COALESCE(city, ''), COALESCE(state, ''), COALESCE(country, 'India'), COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(rating, 4.90), COALESCE(review_count, 0), COALESCE(total_calls_count, 0), COALESCE(total_minutes_spoken, 0), voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg, is_online, is_busy, active_token, created_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Role, &user.AvatarURL, &user.Bio,
		&user.Age, &user.Gender, &user.City, &user.State, &user.Country, &user.Latitude, &user.Longitude,
		&user.Rating, &user.ReviewCount, &user.TotalCallsCount, &user.TotalMinutesSpoken,
		&user.VoiceRatePerMin, &user.VideoRatePerMin, &user.GroupRatePerMin, &user.ChatRatePerMsg, &user.IsOnline, &user.IsBusy, &user.ActiveToken, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}

func (r *userRepo) ListModels() ([]*domain.User, error) {
	rows, err := r.db.Query(`
		SELECT id, phone, name, role, avatar_url, bio, COALESCE(age, 21), COALESCE(gender, 'female'), COALESCE(city, ''), COALESCE(state, ''), COALESCE(country, 'India'), COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(rating, 4.90), COALESCE(review_count, 0), COALESCE(total_calls_count, 0), COALESCE(total_minutes_spoken, 0), voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg, is_online, is_busy, created_at
		FROM users WHERE role = 'model' ORDER BY is_online DESC, rating DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.Name, &u.Role, &u.AvatarURL, &u.Bio,
			&u.Age, &u.Gender, &u.City, &u.State, &u.Country, &u.Latitude, &u.Longitude,
			&u.Rating, &u.ReviewCount, &u.TotalCallsCount, &u.TotalMinutesSpoken,
			&u.VoiceRatePerMin, &u.VideoRatePerMin, &u.GroupRatePerMin, &u.ChatRatePerMsg, &u.IsOnline, &u.IsBusy, &u.CreatedAt,
		); err == nil {
			list = append(list, &u)
		}
	}
	return list, nil
}

func (r *userRepo) ListOnlineUsers() ([]*domain.User, error) {
	rows, err := r.db.Query(`
		SELECT id, phone, name, role, avatar_url, bio, COALESCE(age, 21), COALESCE(gender, 'male'), COALESCE(city, 'New Delhi'), COALESCE(state, 'Delhi'), COALESCE(country, 'India'), COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(rating, 5.0), COALESCE(review_count, 0), COALESCE(total_calls_count, 0), COALESCE(total_minutes_spoken, 0), voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg, is_online, is_busy, created_at
		FROM users WHERE role = 'user' ORDER BY is_online DESC, created_at DESC LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.Name, &u.Role, &u.AvatarURL, &u.Bio,
			&u.Age, &u.Gender, &u.City, &u.State, &u.Country, &u.Latitude, &u.Longitude,
			&u.Rating, &u.ReviewCount, &u.TotalCallsCount, &u.TotalMinutesSpoken,
			&u.VoiceRatePerMin, &u.VideoRatePerMin, &u.GroupRatePerMin, &u.ChatRatePerMsg, &u.IsOnline, &u.IsBusy, &u.CreatedAt,
		); err == nil {
			list = append(list, &u)
		}
	}
	return list, nil
}

func (r *userRepo) DeleteMockModels() error {
	_, err := r.db.Exec(`
		DELETE FROM model_profiles WHERE user_id IN ('model-1', 'model-2', 'model-3', 'model-4', 'user-1', 'user-2', 'user-3');
		DELETE FROM wallets WHERE user_id IN ('model-1', 'model-2', 'model-3', 'model-4', 'user-1', 'user-2', 'user-3');
		DELETE FROM users WHERE id IN ('model-1', 'model-2', 'model-3', 'model-4', 'user-1', 'user-2', 'user-3');
	`)
	return err
}

func (r *userRepo) ListModelsAdvanced(filter *domain.ModelFilterParams) ([]*domain.ModelItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	var query strings.Builder
	var args []interface{}
	argIdx := 1

	hasGeo := filter.Latitude != 0 && filter.Longitude != 0

	query.WriteString(`
		SELECT 
			u.id, u.phone, u.name, u.role, u.avatar_url, u.bio, 
			COALESCE(u.age, 21), COALESCE(u.gender, 'female'), COALESCE(u.city, ''), COALESCE(u.state, ''), COALESCE(u.country, 'India'),
			COALESCE(u.latitude, 0), COALESCE(u.longitude, 0),
			COALESCE(u.rating, 4.90), COALESCE(u.review_count, 0), COALESCE(u.total_calls_count, 0), COALESCE(u.total_minutes_spoken, 0),
			u.voice_rate_per_min, COALESCE(u.video_rate_per_min, 20.0), u.group_rate_per_min, u.chat_rate_per_msg,
			u.is_online, u.is_busy, u.created_at,
			COALESCE(p.display_name, u.name),
			COALESCE(p.languages, 'English, Hindi'),
			COALESCE(p.interests, 'Conversations, Music'),
			COALESCE(p.gallery_urls, ''),
			COALESCE(p.audio_intro_url, ''),
			(CASE WHEN p.status = 'approved' THEN true ELSE false END) AS profile_verified,
	`)

	if hasGeo {
		query.WriteString(fmt.Sprintf(`
			(6371 * acos(least(1.0, greatest(-1.0, 
				cos(radians($%d)) * cos(radians(COALESCE(u.latitude, 28.6139))) * 
				cos(radians(COALESCE(u.longitude, 77.2090)) - radians($%d)) + 
				sin(radians($%d)) * sin(radians(COALESCE(u.latitude, 28.6139)))
			)))) AS distance_km,
		`, argIdx, argIdx+1, argIdx))
		args = append(args, filter.Latitude, filter.Longitude)
		argIdx += 2
	} else {
		query.WriteString(`NULL::numeric AS distance_km, `)
	}

	query.WriteString(`COUNT(*) OVER() AS total_count `)
	query.WriteString(`FROM users u LEFT JOIN model_profiles p ON u.id = p.user_id WHERE u.role = 'model' `)

	if filter.IsOnline != nil && *filter.IsOnline {
		query.WriteString(fmt.Sprintf(`AND u.is_online = $%d `, argIdx))
		args = append(args, true)
		argIdx++
	}

	if filter.MinAge > 0 {
		query.WriteString(fmt.Sprintf(`AND COALESCE(u.age, 21) >= $%d `, argIdx))
		args = append(args, filter.MinAge)
		argIdx++
	}

	if filter.MaxAge > 0 {
		query.WriteString(fmt.Sprintf(`AND COALESCE(u.age, 21) <= $%d `, argIdx))
		args = append(args, filter.MaxAge)
		argIdx++
	}

	if filter.Gender != "" && filter.Gender != "all" {
		query.WriteString(fmt.Sprintf(`AND LOWER(COALESCE(u.gender, 'female')) = LOWER($%d) `, argIdx))
		args = append(args, filter.Gender)
		argIdx++
	}

	if filter.City != "" {
		query.WriteString(fmt.Sprintf(`AND LOWER(COALESCE(u.city, '')) LIKE LOWER($%d) `, argIdx))
		args = append(args, "%"+filter.City+"%")
		argIdx++
	}

	if filter.Language != "" {
		query.WriteString(fmt.Sprintf(`AND LOWER(COALESCE(p.languages, '')) LIKE LOWER($%d) `, argIdx))
		args = append(args, "%"+filter.Language+"%")
		argIdx++
	}

	if filter.Interest != "" {
		query.WriteString(fmt.Sprintf(`AND LOWER(COALESCE(p.interests, '')) LIKE LOWER($%d) `, argIdx))
		args = append(args, "%"+filter.Interest+"%")
		argIdx++
	}

	if filter.MinRate > 0 {
		query.WriteString(fmt.Sprintf(`AND u.voice_rate_per_min >= $%d `, argIdx))
		args = append(args, filter.MinRate)
		argIdx++
	}

	if filter.MaxRate > 0 {
		query.WriteString(fmt.Sprintf(`AND u.voice_rate_per_min <= $%d `, argIdx))
		args = append(args, filter.MaxRate)
		argIdx++
	}

	// Sorting and Filter Modes
	switch filter.Filter {
	case "nearby":
		query.WriteString(`ORDER BY distance_km ASC NULLS LAST, u.is_online DESC, u.rating DESC `)
	case "new":
		query.WriteString(`ORDER BY u.created_at DESC `)
	case "top":
		query.WriteString(`ORDER BY u.rating DESC, u.total_calls_count DESC `)
	case "online":
		query.WriteString(`ORDER BY u.is_online DESC, u.is_busy ASC, u.rating DESC `)
	default:
		switch filter.SortBy {
		case "distance":
			query.WriteString(`ORDER BY distance_km ASC NULLS LAST, u.is_online DESC `)
		case "rating":
			query.WriteString(`ORDER BY u.rating DESC, u.review_count DESC `)
		case "newest":
			query.WriteString(`ORDER BY u.created_at DESC `)
		case "calls", "popularity":
			query.WriteString(`ORDER BY u.total_calls_count DESC, u.rating DESC `)
		case "price_low":
			query.WriteString(`ORDER BY u.voice_rate_per_min ASC `)
		case "price_high":
			query.WriteString(`ORDER BY u.voice_rate_per_min DESC `)
		default:
			query.WriteString(`ORDER BY u.is_online DESC, u.rating DESC, u.created_at DESC `)
		}
	}

	query.WriteString(fmt.Sprintf(`LIMIT $%d OFFSET $%d`, argIdx, argIdx+1))
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*domain.ModelItem
	totalCount := 0

	for rows.Next() {
		var (
			it           domain.ModelItem
			languagesStr string
			interestsStr string
			galleryStr   string
			audioIntro   string
			distNullable sql.NullFloat64
			verified     bool
			total        int
		)
		err := rows.Scan(
			&it.ID, &it.Phone, &it.Name, &it.Role, &it.AvatarURL, &it.Bio,
			&it.Age, &it.Gender, &it.City, &it.State, &it.Country,
			&it.Latitude, &it.Longitude,
			&it.Rating, &it.ReviewCount, &it.TotalCallsCount, &it.TotalMinutesSpoken,
			&it.VoiceRatePerMin, &it.VideoRatePerMin, &it.GroupRatePerMin, &it.ChatRatePerMsg,
			&it.IsOnline, &it.IsBusy, &it.CreatedAt,
			&it.DisplayName, &languagesStr, &interestsStr, &galleryStr, &audioIntro,
			&verified, &distNullable, &total,
		)
		if err != nil {
			continue
		}

		totalCount = total
		it.AudioIntroURL = audioIntro
		it.ProfileVerified = verified

		// Parse comma separated languages and interests
		if languagesStr != "" {
			for _, part := range strings.Split(languagesStr, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					it.Languages = append(it.Languages, p)
				}
			}
		}
		if interestsStr != "" {
			for _, part := range strings.Split(interestsStr, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					it.Interests = append(it.Interests, p)
				}
			}
		}
		if galleryStr != "" {
			for _, part := range strings.Split(galleryStr, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					it.GalleryURLs = append(it.GalleryURLs, p)
				}
			}
		}

		// Distance
		if distNullable.Valid {
			d := distNullable.Float64
			it.DistanceKM = &d
		}

		// Badges
		if time.Since(it.CreatedAt) < 14*24*time.Hour {
			it.IsNew = true
			it.Badges = append(it.Badges, "New Creator")
		}
		if it.Rating >= 4.90 {
			it.Badges = append(it.Badges, "Top Rated")
		}
		if it.TotalCallsCount >= 100 {
			it.Badges = append(it.Badges, "Popular")
		}
		if verified {
			it.Badges = append(it.Badges, "Verified")
		}
		if it.DistanceKM != nil && *it.DistanceKM <= 25.0 {
			it.Badges = append(it.Badges, "Nearby")
		}

		items = append(items, &it)
	}

	return items, totalCount, nil
}

func (r *userRepo) UpdateUserOnboarding(userID string, p *domain.ModelProfile) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET name = COALESCE(NULLIF($1, ''), name), bio = COALESCE(NULLIF($2, ''), bio), avatar_url = COALESCE(NULLIF($3, ''), avatar_url),
		    age = CASE WHEN $4 > 0 THEN $4 ELSE age END,
		    gender = COALESCE(NULLIF($5, ''), gender),
		    city = COALESCE(NULLIF($6, ''), city),
		    state = COALESCE(NULLIF($7, ''), state),
		    country = COALESCE(NULLIF($8, ''), country),
		    latitude = CASE WHEN $9 != 0 THEN $9 ELSE latitude END,
		    longitude = CASE WHEN $10 != 0 THEN $10 ELSE longitude END,
		    voice_rate_per_min = CASE WHEN $11 > 0 THEN $11 ELSE voice_rate_per_min END,
		    video_rate_per_min = CASE WHEN $12 > 0 THEN $12 ELSE video_rate_per_min END,
		    group_rate_per_min = CASE WHEN $13 > 0 THEN $13 ELSE group_rate_per_min END,
		    chat_rate_per_msg = CASE WHEN $14 > 0 THEN $14 ELSE chat_rate_per_msg END
		WHERE id = $15
	`, p.DisplayName, p.Bio, p.AvatarURL, p.Age, p.Gender, p.City, p.State, p.Country,
		p.Latitude, p.Longitude, p.VoiceRatePerMin, p.VideoRatePerMin, p.GroupRatePerMin, p.ChatRatePerMsg, userID)
	return err
}

func (r *userRepo) SetPresence(id string, isOnline, isBusy bool) error {
	_, err := r.db.Exec(`UPDATE users SET is_online = $1, is_busy = $2 WHERE id = $3`, isOnline, isBusy, id)
	return err
}

// ----------------- WALLET REPO -----------------
type walletRepo struct{ db *sql.DB }

func (r *walletRepo) GetWallet(userID string) (*domain.Wallet, error) {
	var w domain.Wallet
	err := r.db.QueryRow(`
		SELECT user_id, balance, bonus_given, total_spent, total_earned, updated_at
		FROM wallets WHERE user_id = $1
	`, userID).Scan(&w.UserID, &w.Balance, &w.BonusGiven, &w.TotalSpent, &w.TotalEarned, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *walletRepo) GetTransactions(userID string) ([]*domain.Transaction, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, amount, type, description, COALESCE(call_id, ''), COALESCE(room_id, ''), created_at
		FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Type, &tx.Description, &tx.CallID, &tx.RoomID, &tx.CreatedAt); err == nil {
			list = append(list, &tx)
		}
	}
	return list, nil
}

func (r *walletRepo) Recharge(userID string, amount float64) (*domain.Wallet, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var w domain.Wallet
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
	`, uuid.New().String(), userID, amount, domain.TxTypeRecharge, fmt.Sprintf("Wallet Recharge of ₹%.2f", amount))
	if err != nil {
		return nil, err
	}

	return &w, tx.Commit()
}

func (r *walletRepo) ProcessCallSettlement(callerID, receiverID, callID string, durationSec int, ratePerMin float64, reason string) (float64, error) {
	if durationSec <= 0 {
		return 0, nil
	}

	cost := (float64(durationSec) / 60.0) * ratePerMin
	tx, err := r.db.Begin()
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

	_, err = tx.Exec(`UPDATE wallets SET balance = balance - $1, total_spent = total_spent + $1, updated_at = NOW() WHERE user_id = $2`, cost, callerID)
	if err != nil {
		return 0, err
	}

	modelShare := cost * 0.8
	_, err = tx.Exec(`UPDATE wallets SET balance = balance + $1, total_earned = total_earned + $1, updated_at = NOW() WHERE user_id = $2`, modelShare, receiverID)
	if err != nil {
		return 0, err
	}

	_, _ = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, call_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New().String(), callerID, -cost, domain.TxTypeCallDebit, fmt.Sprintf("Voice Call (%ds @ ₹%.1f/min) - %s", durationSec, ratePerMin, reason), callID)

	_, _ = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, call_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New().String(), receiverID, modelShare, domain.TxTypeCallCredit, fmt.Sprintf("Call Earnings (%ds @ ₹%.1f/min)", durationSec, ratePerMin), callID)

	return cost, tx.Commit()
}

func (r *walletRepo) DeductChatFee(callerID, receiverID string, amount float64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	err = tx.QueryRow(`SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE`, callerID).Scan(&currentBalance)
	if err != nil || currentBalance < amount {
		return fmt.Errorf("insufficient balance for chat message")
	}

	_, err = tx.Exec(`UPDATE wallets SET balance = balance - $1, total_spent = total_spent + $1, updated_at = NOW() WHERE user_id = $2`, amount, callerID)
	if err != nil {
		return err
	}

	modelShare := amount * 0.8
	_, _ = tx.Exec(`UPDATE wallets SET balance = balance + $1, total_earned = total_earned + $1, updated_at = NOW() WHERE user_id = $2`, modelShare, receiverID)

	_, _ = tx.Exec(`
		INSERT INTO transactions (id, user_id, amount, type, description, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New().String(), callerID, -amount, domain.TxTypeChatDebit, fmt.Sprintf("Chat Message to %s", receiverID))

	return tx.Commit()
}

// ----------------- CALL REPO -----------------
type callRepo struct {
	db      *sql.DB
	wallets repository.WalletRepository
}

func (r *callRepo) Create(c *domain.CallRecord) error {
	_, err := r.db.Exec(`
		INSERT INTO call_records (id, caller_id, caller_name, receiver_id, receiver_name, call_type, status, rate_per_min, started_at, ended_at, last_heartbeat, duration_seconds, total_cost, end_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), $11, $12, $13, $14)
	`, c.ID, c.CallerID, c.CallerName, c.ReceiverID, c.ReceiverName, c.CallType, c.Status, c.RatePerMin, c.StartedAt, c.EndedAt, c.DurationSeconds, c.TotalCost, c.EndReason, c.CreatedAt)
	return err
}

func (r *callRepo) Update(c *domain.CallRecord) error {
	_, err := r.db.Exec(`
		UPDATE call_records
		SET status = $1, started_at = $2, ended_at = $3, duration_seconds = $4, total_cost = $5, end_reason = $6, last_heartbeat = NOW()
		WHERE id = $7
	`, c.Status, c.StartedAt, c.EndedAt, c.DurationSeconds, c.TotalCost, c.EndReason, c.ID)
	return err
}

func (r *callRepo) GetByID(id string) (*domain.CallRecord, error) {
	var c domain.CallRecord
	err := r.db.QueryRow(`
		SELECT id, caller_id, caller_name, receiver_id, receiver_name, call_type, status, rate_per_min, started_at, ended_at, duration_seconds, total_cost, COALESCE(end_reason, ''), created_at
		FROM call_records WHERE id = $1
	`, id).Scan(
		&c.ID, &c.CallerID, &c.CallerName, &c.ReceiverID, &c.ReceiverName, &c.CallType,
		&c.Status, &c.RatePerMin, &c.StartedAt, &c.EndedAt, &c.DurationSeconds, &c.TotalCost, &c.EndReason, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *callRepo) GetUserHistory(userID string) ([]*domain.CallRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, caller_id, caller_name, receiver_id, receiver_name, call_type, status, rate_per_min, started_at, ended_at, duration_seconds, total_cost, COALESCE(end_reason, ''), created_at
		FROM call_records WHERE caller_id = $1 OR receiver_id = $1 ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.CallRecord
	for rows.Next() {
		var c domain.CallRecord
		if err := rows.Scan(
			&c.ID, &c.CallerID, &c.CallerName, &c.ReceiverID, &c.ReceiverName, &c.CallType,
			&c.Status, &c.RatePerMin, &c.StartedAt, &c.EndedAt, &c.DurationSeconds, &c.TotalCost, &c.EndReason, &c.CreatedAt,
		); err == nil {
			list = append(list, &c)
		}
	}
	return list, nil
}

func (r *callRepo) UpdateHeartbeat(callID string) error {
	_, err := r.db.Exec(`UPDATE call_records SET last_heartbeat = NOW() WHERE id = $1 AND status = 'active'`, callID)
	return err
}

func (r *callRepo) RecoverInterruptedCalls() error {
	rows, err := r.db.Query(`
		SELECT id, caller_id, receiver_id, rate_per_min, started_at, last_heartbeat
		FROM call_records WHERE status = 'active'
	`)
	if err != nil {
		return err
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

			if startedAt != nil {
				if lastHeartbeat != nil && lastHeartbeat.After(*startedAt) {
					durationSec = int(lastHeartbeat.Sub(*startedAt).Seconds())
					endedAt = *lastHeartbeat
				} else {
					durationSec = 0
					endedAt = *startedAt
				}
			}

			cost, _ := r.wallets.ProcessCallSettlement(callerID, receiverID, id, durationSec, ratePerMin, "Call cut during server interruption")

			_, _ = r.db.Exec(`
				UPDATE call_records
				SET status = 'completed', ended_at = $1, duration_seconds = $2, total_cost = $3, end_reason = 'Call cut during server interruption'
				WHERE id = $4
			`, endedAt, durationSec, cost, id)

			_, _ = r.db.Exec(`UPDATE users SET is_busy = FALSE WHERE id = $1`, receiverID)
			count++
		}
	}

	if count > 0 {
		log.Printf("🔄 Server Crash Recovery: Reconciled %d calls from heartbeat checkpoints.", count)
	}
	return nil
}

// ----------------- ROOM REPO -----------------
type roomRepo struct {
	db      *sql.DB
	wallets repository.WalletRepository
}

func (r *roomRepo) Create(host *domain.User, title string, ratePerMin float64) (*domain.GroupRoom, error) {
	roomID := "room_" + uuid.New().String()[:8]
	room := &domain.GroupRoom{
		ID:              roomID,
		HostID:          host.ID,
		HostName:        host.Name,
		HostAvatar:      host.AvatarURL,
		Title:           title,
		RatePerMin:      ratePerMin,
		MaxParticipants: 10,
		IsActive:        true,
		Participants:    make(map[string]*domain.RoomParticipant),
		CreatedAt:       time.Now(),
	}

	_, err := r.db.Exec(`
		INSERT INTO group_rooms (id, host_id, host_name, host_avatar, title, rate_per_min, max_participants, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, room.ID, room.HostID, room.HostName, room.HostAvatar, room.Title, room.RatePerMin, room.MaxParticipants, room.IsActive, room.CreatedAt)
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *roomRepo) GetByID(roomID string) (*domain.GroupRoom, error) {
	var room domain.GroupRoom
	err := r.db.QueryRow(`
		SELECT id, host_id, host_name, host_avatar, title, rate_per_min, max_participants, is_active, created_at
		FROM group_rooms WHERE id = $1 AND is_active = TRUE
	`, roomID).Scan(&room.ID, &room.HostID, &room.HostName, &room.HostAvatar, &room.Title, &room.RatePerMin, &room.MaxParticipants, &room.IsActive, &room.CreatedAt)
	if err != nil {
		return nil, err
	}
	room.Participants = make(map[string]*domain.RoomParticipant)
	return &room, nil
}

func (r *roomRepo) ListActive() ([]*domain.GroupRoom, error) {
	rows, err := r.db.Query(`SELECT id, host_id, host_name, host_avatar, title, rate_per_min, max_participants, is_active, created_at FROM group_rooms WHERE is_active = TRUE ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.GroupRoom
	for rows.Next() {
		var rm domain.GroupRoom
		if err := rows.Scan(&rm.ID, &rm.HostID, &rm.HostName, &rm.HostAvatar, &rm.Title, &rm.RatePerMin, &rm.MaxParticipants, &rm.IsActive, &rm.CreatedAt); err == nil {
			rm.Participants = make(map[string]*domain.RoomParticipant)
			list = append(list, &rm)
		}
	}
	return list, nil
}

func (r *roomRepo) AddParticipant(roomID string, user *domain.User) (*domain.RoomParticipant, error) {
	return &domain.RoomParticipant{UserID: user.ID, Name: user.Name, AvatarURL: user.AvatarURL, JoinedAt: time.Now(), IsHost: false}, nil
}

func (r *roomRepo) RemoveParticipant(roomID, userID, reason string) (float64, int, error) {
	return 0, 0, nil
}

// ----------------- MESSAGE REPO -----------------
type messageRepo struct{ db *sql.DB }

func (r *messageRepo) Save(msg *domain.EphemeralMessage) error {
	_, err := r.db.Exec(`
		INSERT INTO ephemeral_messages (id, sender_id, receiver_id, room_id, content, cost, expires_at, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, msg.ID, msg.SenderID, msg.ReceiverID, msg.RoomID, msg.Content, msg.Cost, msg.ExpiresAt, msg.IsRead, msg.CreatedAt)
	return err
}

func (r *messageRepo) GetActive(u1, u2 string) ([]*domain.EphemeralMessage, error) {
	rows, err := r.db.Query(`
		SELECT id, sender_id, receiver_id, room_id, content, cost, expires_at, is_read, created_at
		FROM ephemeral_messages
		WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)) AND expires_at > NOW()
	`, u1, u2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.EphemeralMessage
	for rows.Next() {
		var m domain.EphemeralMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.RoomID, &m.Content, &m.Cost, &m.ExpiresAt, &m.IsRead, &m.CreatedAt); err == nil {
			list = append(list, &m)
		}
	}
	return list, nil
}

func (r *messageRepo) GetConversations(userID string) ([]*dto.ConversationDTO, error) {
	rows, err := r.db.Query(`
		WITH ranked_messages AS (
			SELECT 
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END AS partner_id,
				content,
				created_at,
				is_read,
				sender_id,
				ROW_NUMBER() OVER (
					PARTITION BY (CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END)
					ORDER BY created_at DESC
				) as rn
			FROM ephemeral_messages
			WHERE (sender_id = $1 OR receiver_id = $1) AND expires_at > NOW()
		)
		SELECT 
			m.partner_id,
			COALESCE(u.name, 'User') as partner_name,
			COALESCE(u.avatar_url, '') as partner_avatar,
			m.content as last_message,
			EXTRACT(EPOCH FROM m.created_at)*1000 as last_message_time,
			COALESCE(u.is_online, false) as is_online,
			COALESCE((SELECT COUNT(*) FROM ephemeral_messages em WHERE em.receiver_id = $1 AND em.sender_id = m.partner_id AND em.is_read = FALSE AND em.expires_at > NOW()), 0) as unread_count
		FROM ranked_messages m
		LEFT JOIN users u ON u.id = m.partner_id
		WHERE m.rn = 1
		ORDER BY m.created_at DESC;
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*dto.ConversationDTO
	for rows.Next() {
		var c dto.ConversationDTO
		var lastTime float64
		if err := rows.Scan(&c.PartnerID, &c.PartnerName, &c.PartnerAvatar, &c.LastMessage, &lastTime, &c.IsOnline, &c.UnreadCount); err == nil {
			c.ID = "conv_" + c.PartnerID
			c.LastMessageTime = int64(lastTime)
			list = append(list, &c)
		}
	}
	return list, nil
}

func (r *messageRepo) PurgeExpired() error {
	_, err := r.db.Exec(`DELETE FROM ephemeral_messages WHERE expires_at <= NOW()`)
	return err
}

// ----------------- PAYMENT REPO -----------------
type paymentRepo struct{ db *sql.DB }

func (r *paymentRepo) CreateOrder(order *domain.PaymentOrder) error {
	_, err := r.db.Exec(`
		INSERT INTO payment_orders (id, original_payment_id, user_id, amount, currency, status, gateway_name, gateway_order_id, gateway_payment_id, gateway_signature, failure_reason, refund_reason, refund_id, retry_count, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, order.ID, order.OriginalPaymentID, order.UserID, order.Amount, order.Currency, order.Status, order.GatewayName,
		order.GatewayOrderID, order.GatewayPaymentID, order.GatewaySignature, order.FailureReason, order.RefundReason,
		order.RefundID, order.RetryCount, order.CreatedAt, order.UpdatedAt, order.CompletedAt)
	return err
}

func (r *paymentRepo) UpdateOrder(order *domain.PaymentOrder) error {
	order.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE payment_orders
		SET status = $1, gateway_payment_id = $2, gateway_signature = $3, failure_reason = $4, refund_reason = $5, refund_id = $6, retry_count = $7, updated_at = $8, completed_at = $9
		WHERE id = $10
	`, order.Status, order.GatewayPaymentID, order.GatewaySignature, order.FailureReason, order.RefundReason, order.RefundID, order.RetryCount, order.UpdatedAt, order.CompletedAt, order.ID)
	return err
}

func (r *paymentRepo) GetOrderByID(paymentID string) (*domain.PaymentOrder, error) {
	var o domain.PaymentOrder
	var origID, gwOrdID, gwPayID, gwSig, failReason, refReason, refID sql.NullString
	err := r.db.QueryRow(`
		SELECT id, original_payment_id, user_id, amount, currency, status, gateway_name, gateway_order_id, gateway_payment_id, gateway_signature, failure_reason, refund_reason, refund_id, retry_count, created_at, updated_at, completed_at
		FROM payment_orders WHERE id = $1
	`, paymentID).Scan(
		&o.ID, &origID, &o.UserID, &o.Amount, &o.Currency, &o.Status, &o.GatewayName,
		&gwOrdID, &gwPayID, &gwSig, &failReason, &refReason, &refID, &o.RetryCount,
		&o.CreatedAt, &o.UpdatedAt, &o.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("payment order not found: %w", err)
	}
	o.OriginalPaymentID = origID.String
	o.GatewayOrderID = gwOrdID.String
	o.GatewayPaymentID = gwPayID.String
	o.GatewaySignature = gwSig.String
	o.FailureReason = failReason.String
	o.RefundReason = refReason.String
	o.RefundID = refID.String
	return &o, nil
}

func (r *paymentRepo) GetOrdersByUserID(userID string) ([]*domain.PaymentOrder, error) {
	rows, err := r.db.Query(`
		SELECT id, original_payment_id, user_id, amount, currency, status, gateway_name, gateway_order_id, gateway_payment_id, gateway_signature, failure_reason, refund_reason, refund_id, retry_count, created_at, updated_at, completed_at
		FROM payment_orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.PaymentOrder
	for rows.Next() {
		var o domain.PaymentOrder
		var origID, gwOrdID, gwPayID, gwSig, failReason, refReason, refID sql.NullString
		if err := rows.Scan(
			&o.ID, &origID, &o.UserID, &o.Amount, &o.Currency, &o.Status, &o.GatewayName,
			&gwOrdID, &gwPayID, &gwSig, &failReason, &refReason, &refID, &o.RetryCount,
			&o.CreatedAt, &o.UpdatedAt, &o.CompletedAt,
		); err == nil {
			o.OriginalPaymentID = origID.String
			o.GatewayOrderID = gwOrdID.String
			o.GatewayPaymentID = gwPayID.String
			o.GatewaySignature = gwSig.String
			o.FailureReason = failReason.String
			o.RefundReason = refReason.String
			o.RefundID = refID.String
			list = append(list, &o)
		}
	}
	return list, nil
}

func (r *paymentRepo) RecordAuditLog(log *domain.PaymentAuditLog) error {
	_, err := r.db.Exec(`
		INSERT INTO payment_audit_logs (id, payment_id, from_status, to_status, event_name, gateway_ref_id, gateway_code, message, metadata_json, duration_ms, started_at, ended_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, log.ID, log.PaymentID, log.FromStatus, log.ToStatus, log.EventName, log.GatewayRefID, log.GatewayCode, log.Message, log.MetadataJSON, log.DurationMS, log.StartedAt, log.EndedAt, log.CreatedAt)
	return err
}

func (r *paymentRepo) GetAuditLogs(paymentID string) ([]*domain.PaymentAuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, payment_id, from_status, to_status, event_name, COALESCE(gateway_ref_id, ''), COALESCE(gateway_code, ''), message, COALESCE(metadata_json, ''), duration_ms, started_at, ended_at, created_at
		FROM payment_audit_logs WHERE payment_id = $1 ORDER BY created_at ASC
	`, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.PaymentAuditLog
	for rows.Next() {
		var l domain.PaymentAuditLog
		if err := rows.Scan(&l.ID, &l.PaymentID, &l.FromStatus, &l.ToStatus, &l.EventName, &l.GatewayRefID, &l.GatewayCode, &l.Message, &l.MetadataJSON, &l.DurationMS, &l.StartedAt, &l.EndedAt, &l.CreatedAt); err == nil {
			list = append(list, &l)
		}
	}
	return list, nil
}

// ----------------- MODEL ONBOARDING REPO -----------------
type modelOnboardingRepo struct{ db *sql.DB }

func (r *modelOnboardingRepo) SaveProfile(p *domain.ModelProfile) error {
	galleryStr := strings.Join(p.GalleryURLs, ",")
	if p.SafetyAcceptedAt == nil {
		now := time.Now()
		p.SafetyAcceptedAt = &now
	}

	_, err := r.db.Exec(`
		INSERT INTO model_profiles (
			id, user_id, full_legal_name, display_name, bio, avatar_url, gallery_urls,
			date_of_birth, age, gender, govt_id_type, govt_id_number, govt_id_doc_url, selfie_verification_url,
			city, state, country, pincode, address_line, latitude, longitude,
			languages, interests, voice_rate_per_min, video_rate_per_min, group_rate_per_min, chat_rate_per_msg,
			payout_method, payout_upi, payout_bank_acc, payout_ifsc, payout_beneficiary_name, pan_number,
			audio_intro_url, status, rejection_reason, agreed_to_safety_guidelines, agreed_to_terms, safety_accepted_at,
			report_count, is_suspended, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26, $27,
			$28, $29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38, $39,
			$40, $41, $42, $43
		)
		ON CONFLICT (user_id) DO UPDATE SET
			full_legal_name = EXCLUDED.full_legal_name,
			display_name = EXCLUDED.display_name, bio = EXCLUDED.bio, avatar_url = EXCLUDED.avatar_url, gallery_urls = EXCLUDED.gallery_urls,
			date_of_birth = EXCLUDED.date_of_birth, age = EXCLUDED.age, gender = EXCLUDED.gender,
			govt_id_type = EXCLUDED.govt_id_type, govt_id_number = EXCLUDED.govt_id_number,
			govt_id_doc_url = EXCLUDED.govt_id_doc_url, selfie_verification_url = EXCLUDED.selfie_verification_url,
			city = EXCLUDED.city, state = EXCLUDED.state, country = EXCLUDED.country, pincode = EXCLUDED.pincode, address_line = EXCLUDED.address_line,
			latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
			languages = EXCLUDED.languages, interests = EXCLUDED.interests,
			voice_rate_per_min = EXCLUDED.voice_rate_per_min, video_rate_per_min = EXCLUDED.video_rate_per_min,
			group_rate_per_min = EXCLUDED.group_rate_per_min, chat_rate_per_msg = EXCLUDED.chat_rate_per_msg,
			payout_method = EXCLUDED.payout_method, payout_upi = EXCLUDED.payout_upi,
			payout_bank_acc = EXCLUDED.payout_bank_acc, payout_ifsc = EXCLUDED.payout_ifsc,
			payout_beneficiary_name = EXCLUDED.payout_beneficiary_name, pan_number = EXCLUDED.pan_number,
			audio_intro_url = EXCLUDED.audio_intro_url, status = EXCLUDED.status, updated_at = NOW()
	`,
		p.ID, p.UserID, p.FullLegalName, p.DisplayName, p.Bio, p.AvatarURL, galleryStr,
		p.DateOfBirth, p.Age, p.Gender, p.GovtIDType, p.GovtIDNumber, p.GovtIDDocURL, p.SelfieVerificationURL,
		p.City, p.State, p.Country, p.Pincode, p.AddressLine, p.Latitude, p.Longitude,
		p.Languages, p.Interests, p.VoiceRatePerMin, p.VideoRatePerMin, p.GroupRatePerMin, p.ChatRatePerMsg,
		p.PayoutMethod, p.PayoutUPI, p.PayoutBankAcc, p.PayoutIFSC, p.PayoutBeneficiaryName, p.PANNumber,
		p.AudioIntroURL, p.Status, p.RejectionReason, p.AgreedToSafetyGuidelines, p.AgreedToTerms, p.SafetyAcceptedAt,
		p.ReportCount, p.IsSuspended, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *modelOnboardingRepo) UpdateProfile(p *domain.ModelProfile) error {
	p.UpdatedAt = time.Now()
	galleryStr := strings.Join(p.GalleryURLs, ",")

	_, err := r.db.Exec(`
		UPDATE model_profiles
		SET full_legal_name = $1, display_name = $2, bio = $3, avatar_url = $4, gallery_urls = $5,
		    date_of_birth = $6, age = $7, gender = $8, govt_id_type = $9, govt_id_number = $10,
		    govt_id_doc_url = $11, selfie_verification_url = $12, city = $13, state = $14, country = $15,
		    pincode = $16, address_line = $17, latitude = $18, longitude = $19, languages = $20,
		    interests = $21, voice_rate_per_min = $22, video_rate_per_min = $23, group_rate_per_min = $24,
		    chat_rate_per_msg = $25, payout_method = $26, payout_upi = $27, payout_bank_acc = $28,
		    payout_ifsc = $29, payout_beneficiary_name = $30, pan_number = $31, audio_intro_url = $32,
		    status = $33, rejection_reason = $34, report_count = $35, is_suspended = $36, updated_at = $37
		WHERE user_id = $38
	`,
		p.FullLegalName, p.DisplayName, p.Bio, p.AvatarURL, galleryStr,
		p.DateOfBirth, p.Age, p.Gender, p.GovtIDType, p.GovtIDNumber,
		p.GovtIDDocURL, p.SelfieVerificationURL, p.City, p.State, p.Country,
		p.Pincode, p.AddressLine, p.Latitude, p.Longitude, p.Languages,
		p.Interests, p.VoiceRatePerMin, p.VideoRatePerMin, p.GroupRatePerMin,
		p.ChatRatePerMsg, p.PayoutMethod, p.PayoutUPI, p.PayoutBankAcc,
		p.PayoutIFSC, p.PayoutBeneficiaryName, p.PANNumber, p.AudioIntroURL,
		p.Status, p.RejectionReason, p.ReportCount, p.IsSuspended, p.UpdatedAt, p.UserID,
	)
	return err
}

func (r *modelOnboardingRepo) GetProfileByUserID(userID string) (*domain.ModelProfile, error) {
	var p domain.ModelProfile
	var (
		legalName, galleryStr, dob, idType, idNum, idDoc, selfie, city, state, country, pincode, addr,
		payMethod, upi, bankAcc, ifsc, beneName, pan, audioURL, rejReason sql.NullString
		safetyAccepted sql.NullTime
	)

	err := r.db.QueryRow(`
		SELECT 
			id, user_id, full_legal_name, display_name, bio, avatar_url, gallery_urls,
			date_of_birth, age, gender, govt_id_type, govt_id_number, govt_id_doc_url, selfie_verification_url,
			city, state, country, pincode, address_line, COALESCE(latitude, 0), COALESCE(longitude, 0),
			languages, interests, voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg,
			payout_method, payout_upi, payout_bank_acc, payout_ifsc, payout_beneficiary_name, pan_number,
			audio_intro_url, status, rejection_reason, agreed_to_safety_guidelines, agreed_to_terms, safety_accepted_at,
			report_count, is_suspended, created_at, updated_at
		FROM model_profiles WHERE user_id = $1
	`, userID).Scan(
		&p.ID, &p.UserID, &legalName, &p.DisplayName, &p.Bio, &p.AvatarURL, &galleryStr,
		&dob, &p.Age, &p.Gender, &idType, &idNum, &idDoc, &selfie,
		&city, &state, &country, &pincode, &addr, &p.Latitude, &p.Longitude,
		&p.Languages, &p.Interests, &p.VoiceRatePerMin, &p.VideoRatePerMin, &p.GroupRatePerMin, &p.ChatRatePerMsg,
		&payMethod, &upi, &bankAcc, &ifsc, &beneName, &pan,
		&audioURL, &p.Status, &rejReason, &p.AgreedToSafetyGuidelines, &p.AgreedToTerms, &safetyAccepted,
		&p.ReportCount, &p.IsSuspended, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("model profile not found: %w", err)
	}

	p.FullLegalName = legalName.String
	p.DateOfBirth = dob.String
	p.GovtIDType = idType.String
	p.GovtIDNumber = idNum.String
	p.GovtIDDocURL = idDoc.String
	p.SelfieVerificationURL = selfie.String
	p.City = city.String
	p.State = state.String
	p.Country = country.String
	p.Pincode = pincode.String
	p.AddressLine = addr.String
	p.PayoutMethod = payMethod.String
	p.PayoutUPI = upi.String
	p.PayoutBankAcc = bankAcc.String
	p.PayoutIFSC = ifsc.String
	p.PayoutBeneficiaryName = beneName.String
	p.PANNumber = pan.String
	p.AudioIntroURL = audioURL.String
	p.RejectionReason = rejReason.String
	if safetyAccepted.Valid {
		p.SafetyAcceptedAt = &safetyAccepted.Time
	}

	if galleryStr.Valid && galleryStr.String != "" {
		for _, part := range strings.Split(galleryStr.String, ",") {
			pStr := strings.TrimSpace(part)
			if pStr != "" {
				p.GalleryURLs = append(p.GalleryURLs, pStr)
			}
		}
	}

	return &p, nil
}

func (r *modelOnboardingRepo) ListPendingProfiles() ([]*domain.ModelProfile, error) {
	rows, err := r.db.Query(`
		SELECT 
			id, user_id, full_legal_name, display_name, bio, avatar_url, gallery_urls,
			date_of_birth, age, gender, govt_id_type, govt_id_number, govt_id_doc_url, selfie_verification_url,
			city, state, country, pincode, address_line, COALESCE(latitude, 0), COALESCE(longitude, 0),
			languages, interests, voice_rate_per_min, COALESCE(video_rate_per_min, 20.0), group_rate_per_min, chat_rate_per_msg,
			payout_method, payout_upi, payout_bank_acc, payout_ifsc, payout_beneficiary_name, pan_number,
			audio_intro_url, status, rejection_reason, agreed_to_safety_guidelines, agreed_to_terms, safety_accepted_at,
			report_count, is_suspended, created_at, updated_at
		FROM model_profiles WHERE status = 'pending_review' ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ModelProfile
	for rows.Next() {
		var p domain.ModelProfile
		var (
			legalName, galleryStr, dob, idType, idNum, idDoc, selfie, city, state, country, pincode, addr,
			payMethod, upi, bankAcc, ifsc, beneName, pan, audioURL, rejReason sql.NullString
			safetyAccepted sql.NullTime
		)
		if err := rows.Scan(
			&p.ID, &p.UserID, &legalName, &p.DisplayName, &p.Bio, &p.AvatarURL, &galleryStr,
			&dob, &p.Age, &p.Gender, &idType, &idNum, &idDoc, &selfie,
			&city, &state, &country, &pincode, &addr, &p.Latitude, &p.Longitude,
			&p.Languages, &p.Interests, &p.VoiceRatePerMin, &p.VideoRatePerMin, &p.GroupRatePerMin, &p.ChatRatePerMsg,
			&payMethod, &upi, &bankAcc, &ifsc, &beneName, &pan,
			&audioURL, &p.Status, &rejReason, &p.AgreedToSafetyGuidelines, &p.AgreedToTerms, &safetyAccepted,
			&p.ReportCount, &p.IsSuspended, &p.CreatedAt, &p.UpdatedAt,
		); err == nil {
			p.FullLegalName = legalName.String
			p.DateOfBirth = dob.String
			p.GovtIDType = idType.String
			p.GovtIDNumber = idNum.String
			p.GovtIDDocURL = idDoc.String
			p.SelfieVerificationURL = selfie.String
			p.City = city.String
			p.State = state.String
			p.Country = country.String
			p.Pincode = pincode.String
			p.AddressLine = addr.String
			p.PayoutMethod = payMethod.String
			p.PayoutUPI = upi.String
			p.PayoutBankAcc = bankAcc.String
			p.PayoutIFSC = ifsc.String
			p.PayoutBeneficiaryName = beneName.String
			p.PANNumber = pan.String
			p.AudioIntroURL = audioURL.String
			p.RejectionReason = rejReason.String
			if safetyAccepted.Valid {
				p.SafetyAcceptedAt = &safetyAccepted.Time
			}
			if galleryStr.Valid && galleryStr.String != "" {
				for _, part := range strings.Split(galleryStr.String, ",") {
					pStr := strings.TrimSpace(part)
					if pStr != "" {
						p.GalleryURLs = append(p.GalleryURLs, pStr)
					}
				}
			}
			list = append(list, &p)
		}
	}
	return list, nil
}

func (r *modelOnboardingRepo) IncrementReportCount(modelID string) (int, error) {
	var newCount int
	err := r.db.QueryRow(`
		INSERT INTO model_profiles (id, user_id, display_name, bio, avatar_url, age, gender, languages, interests, report_count, status, created_at, updated_at)
		VALUES ($1, $2, 'Host', 'Bio', '', 18, 'Other', 'English', '', 1, 'approved', NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET report_count = model_profiles.report_count + 1, updated_at = NOW()
		RETURNING report_count
	`, "prof_"+modelID, modelID).Scan(&newCount)
	return newCount, err
}

func (r *modelOnboardingRepo) SetSuspension(modelID string, isSuspended bool) error {
	_, err := r.db.Exec(`UPDATE model_profiles SET is_suspended = $1, updated_at = NOW() WHERE user_id = $2`, isSuspended, modelID)
	return err
}

// ----------------- REPORT REPOSITORY -----------------
type reportRepo struct{ db *sql.DB }

func (r *reportRepo) CreateReport(report *domain.ModelReport) error {
	_, err := r.db.Exec(`
		INSERT INTO model_reports (id, reporter_id, reporter_name, model_id, model_name, call_id, room_id, category, description, status, admin_action, admin_note, created_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, report.ID, report.ReporterID, report.ReporterName, report.ModelID, report.ModelName,
		report.CallID, report.RoomID, report.Category, report.Description, report.Status,
		report.AdminAction, report.AdminNote, report.CreatedAt, report.ResolvedAt)
	return err
}

func (r *reportRepo) UpdateReport(report *domain.ModelReport) error {
	_, err := r.db.Exec(`
		UPDATE model_reports
		SET status = $1, admin_action = $2, admin_note = $3, resolved_at = $4
		WHERE id = $5
	`, report.Status, report.AdminAction, report.AdminNote, report.ResolvedAt, report.ID)
	return err
}

func (r *reportRepo) GetReportByID(id string) (*domain.ModelReport, error) {
	var rep domain.ModelReport
	var callID, roomID, action, note sql.NullString
	err := r.db.QueryRow(`
		SELECT id, reporter_id, reporter_name, model_id, model_name, call_id, room_id, category, description, status, admin_action, admin_note, created_at, resolved_at
		FROM model_reports WHERE id = $1
	`, id).Scan(
		&rep.ID, &rep.ReporterID, &rep.ReporterName, &rep.ModelID, &rep.ModelName,
		&callID, &roomID, &rep.Category, &rep.Description, &rep.Status, &action, &note,
		&rep.CreatedAt, &rep.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}
	rep.CallID = callID.String
	rep.RoomID = roomID.String
	rep.AdminAction = action.String
	rep.AdminNote = note.String
	return &rep, nil
}

func (r *reportRepo) GetReportsForModel(modelID string) ([]*domain.ModelReport, error) {
	rows, err := r.db.Query(`
		SELECT id, reporter_id, reporter_name, model_id, model_name, call_id, room_id, category, description, status, admin_action, admin_note, created_at, resolved_at
		FROM model_reports WHERE model_id = $1 ORDER BY created_at DESC LIMIT 50
	`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ModelReport
	for rows.Next() {
		var rep domain.ModelReport
		var callID, roomID, action, note sql.NullString
		if err := rows.Scan(
			&rep.ID, &rep.ReporterID, &rep.ReporterName, &rep.ModelID, &rep.ModelName,
			&callID, &roomID, &rep.Category, &rep.Description, &rep.Status, &action, &note,
			&rep.CreatedAt, &rep.ResolvedAt,
		); err == nil {
			rep.CallID = callID.String
			rep.RoomID = roomID.String
			rep.AdminAction = action.String
			rep.AdminNote = note.String
			list = append(list, &rep)
		}
	}
	return list, nil
}

func (r *reportRepo) ListRecentReports() ([]*domain.ModelReport, error) {
	rows, err := r.db.Query(`
		SELECT id, reporter_id, reporter_name, model_id, model_name, call_id, room_id, category, description, status, admin_action, admin_note, created_at, resolved_at
		FROM model_reports ORDER BY created_at DESC LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ModelReport
	for rows.Next() {
		var rep domain.ModelReport
		var callID, roomID, action, note sql.NullString
		if err := rows.Scan(
			&rep.ID, &rep.ReporterID, &rep.ReporterName, &rep.ModelID, &rep.ModelName,
			&callID, &roomID, &rep.Category, &rep.Description, &rep.Status, &action, &note,
			&rep.CreatedAt, &rep.ResolvedAt,
		); err == nil {
			rep.CallID = callID.String
			rep.RoomID = roomID.String
			rep.AdminAction = action.String
			rep.AdminNote = note.String
			list = append(list, &rep)
		}
	}
	return list, rows.Err()
}

// ----------------- FAVORITE REPO (POSTGRESQL) -----------------
type favoriteRepo struct{ db *sql.DB }

func (r *favoriteRepo) ToggleFavorite(userID, modelID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM user_favorites WHERE user_id = $1 AND model_id = $2)
	`, userID, modelID).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		_, err = r.db.Exec(`DELETE FROM user_favorites WHERE user_id = $1 AND model_id = $2`, userID, modelID)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	_, err = r.db.Exec(`
		INSERT INTO user_favorites (user_id, model_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, model_id) DO NOTHING
	`, userID, modelID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *favoriteRepo) IsFavorite(userID, modelID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM user_favorites WHERE user_id = $1 AND model_id = $2)
	`, userID, modelID).Scan(&exists)
	return exists, err
}

func (r *favoriteRepo) GetFavoriteModelIDs(userID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT model_id FROM user_favorites WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (r *favoriteRepo) GetFavoriteModels(userID string) ([]*domain.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.phone, u.name, u.role, u.avatar_url, u.bio,
		       u.voice_rate_per_min, u.group_rate_per_min, u.chat_rate_per_msg,
		       u.is_online, u.is_busy, u.created_at
		FROM users u
		JOIN user_favorites f ON u.id = f.model_id
		WHERE f.user_id = $1 AND u.role = 'model'
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.Name, &u.Role, &u.AvatarURL, &u.Bio,
			&u.VoiceRatePerMin, &u.GroupRatePerMin, &u.ChatRatePerMsg,
			&u.IsOnline, &u.IsBusy, &u.CreatedAt,
		); err == nil {
			list = append(list, &u)
		}
	}
	return list, rows.Err()
}

