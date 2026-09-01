-- =======================================================================
-- Connect Clean Database Wipe & Re-initialization Script
-- WARNING: This deletes all existing users, wallets, calls, chats, and records.
-- =======================================================================

-- 1. DROP ALL EXISTING TABLES & INDEXES
DROP TABLE IF EXISTS model_reports CASCADE;
DROP TABLE IF EXISTS user_favorites CASCADE;
DROP TABLE IF EXISTS payment_audit_logs CASCADE;
DROP TABLE IF EXISTS payment_orders CASCADE;
DROP TABLE IF EXISTS ephemeral_messages CASCADE;
DROP TABLE IF EXISTS room_participants CASCADE;
DROP TABLE IF EXISTS group_rooms CASCADE;
DROP TABLE IF EXISTS call_records CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS wallets CASCADE;
DROP TABLE IF EXISTS model_profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- 2. CREATE USERS TABLE
CREATE TABLE users (
    id VARCHAR(64) PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    avatar_url TEXT,
    bio TEXT,
    age INT DEFAULT 21,
    gender VARCHAR(20) DEFAULT 'female',
    city VARCHAR(100) DEFAULT 'New Delhi',
    state VARCHAR(100) DEFAULT 'Delhi',
    country VARCHAR(100) DEFAULT 'India',
    latitude NUMERIC(10, 6) DEFAULT 28.6139,
    longitude NUMERIC(10, 6) DEFAULT 77.2090,
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
    active_token TEXT,
    device_id VARCHAR(100),
    active_room_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_active_token ON users(active_token);

-- 3. CREATE WALLETS TABLE
CREATE TABLE wallets (
    user_id VARCHAR(64) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
    bonus_given NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_spent NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_earned NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. CREATE TRANSACTIONS TABLE
CREATE TABLE transactions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    call_id VARCHAR(64),
    room_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);

-- 5. CREATE CALL RECORDS TABLE
CREATE TABLE call_records (
    id VARCHAR(64) PRIMARY KEY,
    caller_id VARCHAR(64) NOT NULL REFERENCES users(id),
    caller_name VARCHAR(100) NOT NULL,
    receiver_id VARCHAR(64) NOT NULL REFERENCES users(id),
    receiver_name VARCHAR(100) NOT NULL,
    call_type VARCHAR(20) NOT NULL DEFAULT 'voice',
    status VARCHAR(20) NOT NULL DEFAULT 'ringing',
    duration_sec INT NOT NULL DEFAULT 0,
    rate_per_min NUMERIC(10, 2) NOT NULL,
    total_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    end_reason VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_call_records_caller ON call_records(caller_id);
CREATE INDEX idx_call_records_receiver ON call_records(receiver_id);

-- 6. CREATE GROUP ROOMS TABLE
CREATE TABLE group_rooms (
    id VARCHAR(64) PRIMARY KEY,
    host_id VARCHAR(64) NOT NULL REFERENCES users(id),
    host_name VARCHAR(100) NOT NULL,
    title VARCHAR(150) NOT NULL,
    rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
    participant_count INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE
);

-- 6.1 CREATE LIVE STREAM AUDIT TABLE (METADATA & PAYMENTS ONLY)
CREATE TABLE IF NOT EXISTS live_stream_records (
    id VARCHAR(64) PRIMARY KEY,
    host_id VARCHAR(64) NOT NULL REFERENCES users(id),
    host_name VARCHAR(100) NOT NULL,
    title VARCHAR(200) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INT NOT NULL DEFAULT 0,
    peak_viewers INT NOT NULL DEFAULT 1,
    total_earned NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_live_streams_host ON live_stream_records(host_id);

-- 7. CREATE ROOM PARTICIPANTS TABLE
CREATE TABLE room_participants (
    id VARCHAR(64) PRIMARY KEY,
    room_id VARCHAR(64) NOT NULL REFERENCES group_rooms(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    duration_sec INT NOT NULL DEFAULT 0,
    total_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    leave_reason VARCHAR(100)
);

-- 8. CREATE EPHEMERAL MESSAGES TABLE
CREATE TABLE ephemeral_messages (
    id VARCHAR(64) PRIMARY KEY,
    sender_id VARCHAR(64) NOT NULL REFERENCES users(id),
    receiver_id VARCHAR(64) NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    fee_deducted NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_ephemeral_receiver ON ephemeral_messages(receiver_id, expires_at);

-- 9. CREATE MODEL PROFILES TABLE
CREATE TABLE model_profiles (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100) NOT NULL,
    bio TEXT,
    avatar_url TEXT,
    age INT DEFAULT 21,
    gender VARCHAR(20) DEFAULT 'female',
    city VARCHAR(100) DEFAULT 'New Delhi',
    state VARCHAR(100) DEFAULT 'Delhi',
    country VARCHAR(100) DEFAULT 'India',
    latitude NUMERIC(10, 6) DEFAULT 28.6139,
    longitude NUMERIC(10, 6) DEFAULT 77.2090,
    languages TEXT DEFAULT 'English, Hindi',
    interests TEXT DEFAULT 'Conversations, Music',
    voice_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 10.00,
    video_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 20.00,
    group_rate_per_min NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
    chat_rate_per_msg NUMERIC(10, 2) NOT NULL DEFAULT 1.00,
    status VARCHAR(30) NOT NULL DEFAULT 'approved',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 10. CREATE USER FAVORITES & REPORTS TABLES
CREATE TABLE user_favorites (
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, model_id)
);

CREATE TABLE model_reports (
    id VARCHAR(64) PRIMARY KEY,
    reporter_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason VARCHAR(50) NOT NULL,
    details TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

