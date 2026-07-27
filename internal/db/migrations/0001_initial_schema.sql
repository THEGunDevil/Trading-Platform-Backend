-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_name VARCHAR(50) NOT NULL DEFAULT '',
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT DEFAULT 'user' CHECK (role IN ('user','agent','admin')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    token_version INT NOT NULL DEFAULT 1,
    is_banned BOOLEAN DEFAULT FALSE,
    ban_reason TEXT,
    ban_until TIMESTAMP,
    is_permanent_ban BOOLEAN DEFAULT FALSE
);

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    object_id UUID,                 
    object_title TEXT,
    type TEXT NOT NULL,             
    title TEXT NOT NULL,            
    message TEXT NOT NULL,          
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_notification_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, event_id)       
);
-- Only add predictions table, reuse balances for USDT
CREATE TABLE predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    coin_id VARCHAR(50) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    amount DECIMAL(30,4) NOT NULL CHECK (amount > 0),
    direction VARCHAR(4) NOT NULL CHECK (direction IN ('up', 'down')),
    duration_seconds INT NOT NULL CHECK (duration_seconds IN (10, 30, 60, 300)),
    start_price DECIMAL(30,4) NOT NULL,
    final_price DECIMAL(30,4),
    payout_rate DECIMAL(5,4) NOT NULL DEFAULT 0.8000,
    status VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'won', 'lost', 'cancelled')),
    profit DECIMAL(30,4) DEFAULT 0,
    payout DECIMAL(30,4) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_predictions_user_id ON predictions(user_id);
CREATE INDEX idx_predictions_status ON predictions(status);
CREATE INDEX idx_predictions_expires ON predictions(expires_at) WHERE status = 'active';
-- Refresh tokens for login sessions (pairs with users.token_version for revocation)
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Per-user, per-asset balances (spot wallet)
CREATE TABLE balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset TEXT NOT NULL,            -- e.g. "USDT", "BTC"
    available NUMERIC(30, 4) NOT NULL DEFAULT 0 CHECK (available >= 0),
    locked NUMERIC(30, 4) NOT NULL DEFAULT 0 CHECK (locked >= 0), -- reserved by open orders
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, asset)
);
CREATE INDEX idx_balances_user_id ON balances(user_id);
CREATE TABLE IF NOT EXISTS platform_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Deposit records (funds arriving)
CREATE TABLE deposits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset TEXT NOT NULL,
    network TEXT NOT NULL,
    amount NUMERIC(30, 4) NOT NULL CHECK (amount > 0),
    tx_hash TEXT,
    confirmations INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'failed')),
    created_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP
);
CREATE INDEX idx_deposits_user_id ON deposits(user_id);

-- Withdrawal requests (funds leaving)
CREATE TABLE withdrawals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset TEXT NOT NULL,
    network TEXT NOT NULL,
    destination_address TEXT NOT NULL,
    amount NUMERIC(30, 4) NOT NULL CHECK (amount > 0),
    fee NUMERIC(30, 4) NOT NULL DEFAULT 0 CHECK (fee >= 0),
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'rejected', 'failed')),
    tx_hash TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);
CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);

-- Orders placed on the trade terminal
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,           -- e.g. "BTCUSDT"
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    order_type TEXT NOT NULL CHECK (order_type IN ('market', 'limit')),
    leverage INT NOT NULL DEFAULT 1 CHECK (leverage >= 1),
    price NUMERIC(30, 4),          -- null for market orders until filled
    quantity NUMERIC(30, 4) NOT NULL CHECK (quantity > 0),
    margin NUMERIC(30, 4) NOT NULL,
    fee NUMERIC(30, 4) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'filled', 'cancelled', 'rejected')),
    created_at TIMESTAMP DEFAULT NOW(),
    filled_at TIMESTAMP
);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);

-- Executed fills against an order (an order can fill in multiple pieces)
CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    price NUMERIC(30, 4) NOT NULL,
    quantity NUMERIC(30, 4) NOT NULL CHECK (quantity > 0),
    fee NUMERIC(30, 4) NOT NULL DEFAULT 0,
    executed_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_trades_user_id ON trades(user_id);
CREATE INDEX idx_trades_order_id ON trades(order_id);

-- Customer support conversations
CREATE TABLE support_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE support_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_agent_id UUID REFERENCES users(id),          -- renamed from agent_id
    subject VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'assigned', 'in_progress', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ
);

-- Recreate indexes if needed
CREATE INDEX idx_support_sessions_user_id ON support_sessions(user_id);
CREATE INDEX idx_support_sessions_assigned_agent ON support_sessions(assigned_agent_id);
CREATE INDEX idx_support_sessions_status ON support_sessions(status);

-- 2. Messages (text and/or image)
CREATE TABLE support_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES support_sessions(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    content TEXT,
    image_url TEXT,
    is_agent BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (content IS NOT NULL OR image_url IS NOT NULL)
);
CREATE INDEX idx_messages_session ON support_messages(session_id);

-- 3. Notifications (with is_expired)
CREATE TABLE session_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES support_sessions(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES users(id),
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_expired BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS support_messages CASCADE;
DROP TABLE IF EXISTS support_conversations CASCADE;
DROP TABLE IF EXISTS session_notifications CASCADE;
DROP TABLE IF EXISTS support_sessions CASCADE;
DROP TABLE IF EXISTS trades CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS withdrawals CASCADE;
DROP TABLE IF EXISTS deposits CASCADE;
DROP TABLE IF EXISTS platform_settings;
DROP TABLE IF EXISTS balances CASCADE;
DROP TABLE IF EXISTS watchlist_items CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS user_notification_status CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS predictions CASCADE;