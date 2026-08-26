-- =======================================================================
-- Connect PostgreSQL Database Schema
-- Strict ACID Ledger, Single-Device Session Enforcement, Zero Audio Storage
-- =======================================================================

-- 1. Users & Models Table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user', -- 'user' or 'model'
    avatar_url TEXT,
    bio TEXT,
    age INT DEFAULT 21,
    gender VARCHAR(20) DEFAULT 'female',
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100) DEFAULT 'India',
    latitude NUMERIC(10, 6),
    longitude NUMERIC(10, 6),
    rating NUMERIC(3, 2) NOT NULL DEFAULT 4.90,
    review_count INT NOT NULL DEFAULT 0,
    total_calls_count INT NOT NULL DEFAULT 0,
    total_minutes_spoken INT NOT NULL DEFAULT 0,
    voice_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 10.00,
    video_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 20.00,
    group_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
    chat_rate_per_msg NUMERIC(10, 2) NOT NULL DEFAULT 1.00,
    is_online BOOLEAN NOT NULL DEFAULT FALSE,
    is_busy BOOLEAN NOT NULL DEFAULT FALSE,
    active_token TEXT, -- Enforces single active device/session
    device_id VARCHAR(100),
    active_room_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for phone lookups & token validation
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_active_token ON users(active_token);

-- 2. Wallets Table (Strict Balance Ledger)
CREATE TABLE IF NOT EXISTS wallets (
    user_id VARCHAR(64) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
    bonus_given NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_spent NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_earned NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Financial Transactions Ledger Table
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL, -- Positive for credit, negative for debit
    type VARCHAR(50) NOT NULL,      -- 'welcome_bonus', 'recharge', 'call_debit', 'call_credit', 'group_call_debit', 'group_call_credit', 'chat_debit'
    description TEXT,
    call_id VARCHAR(64),
    room_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);

-- 4. Call Records Table (METADATA ONLY - ZERO AUDIO RECORDED/STORED)
CREATE TABLE IF NOT EXISTS call_records (
    id VARCHAR(64) PRIMARY KEY,
    caller_id VARCHAR(64) NOT NULL REFERENCES users(id),
    caller_name VARCHAR(100) NOT NULL,
    receiver_id VARCHAR(64) NOT NULL REFERENCES users(id),
    receiver_name VARCHAR(100) NOT NULL,
    call_type VARCHAR(20) NOT NULL DEFAULT 'voice', -- 'voice', 'group_voice', 'video'
    status VARCHAR(30) NOT NULL,                    -- 'initiated', 'active', 'completed', 'rejected', 'busy', 'balance_exhausted'
    rate_per_min NUMERIC(10, 2) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(), -- Checkpoint for crash recovery
    duration_seconds INT NOT NULL DEFAULT 0,
    total_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    end_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_call_records_caller ON call_records(caller_id);
CREATE INDEX IF NOT EXISTS idx_call_records_receiver ON call_records(receiver_id);
CREATE INDEX IF NOT EXISTS idx_call_records_created ON call_records(created_at DESC);

-- 5. Group Audio Rooms Table
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

-- 6. Ephemeral Chat Messages Table (Auto-Expiring with TTL)
CREATE TABLE IF NOT EXISTS ephemeral_messages (
    id VARCHAR(64) PRIMARY KEY,
    sender_id VARCHAR(64) NOT NULL REFERENCES users(id),
    receiver_id VARCHAR(64) REFERENCES users(id),
    room_id VARCHAR(64),
    content TEXT NOT NULL,
    cost NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ephemeral_expires ON ephemeral_messages(expires_at);

-- 7. Payment Orders Table (Stateful Order Life-Cycle & Retries)
CREATE TABLE IF NOT EXISTS payment_orders (
    id VARCHAR(64) PRIMARY KEY,
    original_payment_id VARCHAR(64),                                   -- Link to parent order if this is a retry
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'INR',
    status VARCHAR(30) NOT NULL,                                       -- 'initiated', 'inprogress', 'successful', 'failed', 'refund_inprogress', 'refund_done'
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

-- 8. Payment Audit Logs Table (Step-by-step Execution Tracking)
CREATE TABLE IF NOT EXISTS payment_audit_logs (
    id VARCHAR(64) PRIMARY KEY,
    payment_id VARCHAR(64) NOT NULL REFERENCES payment_orders(id) ON DELETE CASCADE,
    from_status VARCHAR(30) NOT NULL,
    to_status VARCHAR(30) NOT NULL,
    event_name VARCHAR(100) NOT NULL,                                  -- e.g. "ORDER_INITIALIZED", "GATEWAY_REDIRECT", "CALLBACK_SUCCESS", "REFUND_TRIGGERED"
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

-- 9. Model Onboarding Profiles Table (Full Creator Dossier, Verification, Pricing, Payouts)
CREATE TABLE IF NOT EXISTS model_profiles (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_legal_name VARCHAR(150),
    display_name VARCHAR(100) NOT NULL,
    bio TEXT NOT NULL,
    avatar_url TEXT NOT NULL,
    gallery_urls TEXT,
    date_of_birth VARCHAR(30),
    age INT NOT NULL CHECK (age >= 18),
    gender VARCHAR(20) NOT NULL,
    govt_id_type VARCHAR(50),
    govt_id_number VARCHAR(100),
    govt_id_doc_url TEXT,
    selfie_verification_url TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100) DEFAULT 'India',
    pincode VARCHAR(20),
    address_line TEXT,
    latitude NUMERIC(10, 6),
    longitude NUMERIC(10, 6),
    languages VARCHAR(200) NOT NULL,
    interests VARCHAR(200) NOT NULL,
    voice_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 15.00,
    video_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 20.00,
    group_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 7.00,
    chat_rate_per_msg NUMERIC(10, 2) NOT NULL DEFAULT 2.00,
    payout_method VARCHAR(30) DEFAULT 'upi',
    payout_upi VARCHAR(100),
    payout_bank_acc VARCHAR(50),
    payout_ifsc VARCHAR(20),
    payout_beneficiary_name VARCHAR(150),
    pan_number VARCHAR(20),
    audio_intro_url TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending_review', -- 'pending_review', 'approved', 'rejected'
    rejection_reason TEXT,
    agreed_to_safety_guidelines BOOLEAN NOT NULL DEFAULT TRUE,
    agreed_to_terms BOOLEAN NOT NULL DEFAULT TRUE,
    safety_accepted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    report_count INT NOT NULL DEFAULT 0,
    is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_profiles_user ON model_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_model_profiles_status ON model_profiles(status);

-- 10. User Safety & Model Reports Table
CREATE TABLE IF NOT EXISTS model_reports (
    id VARCHAR(64) PRIMARY KEY,
    reporter_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reporter_name VARCHAR(100) NOT NULL,
    model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_name VARCHAR(100) NOT NULL,
    call_id VARCHAR(64),
    room_id VARCHAR(64),
    category VARCHAR(50) NOT NULL,                        -- 'harassment', 'inappropriate_behavior', 'underage_suspicion', 'fraud', 'poor_audio_quality', 'other'
    description TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'submitted',      -- 'submitted', 'under_investigation', 'resolved', 'dismissed'
    admin_action VARCHAR(50) DEFAULT 'none',
    admin_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_model_reports_model ON model_reports(model_id);
CREATE INDEX IF NOT EXISTS idx_model_reports_reporter ON model_reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_model_reports_status ON model_reports(status);

-- 11. User Favorites Table (Model Bookmarking)
CREATE TABLE IF NOT EXISTS user_favorites (
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_user_favorites_user ON user_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorites_model ON user_favorites(model_id);

