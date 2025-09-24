CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(50),
    first_name VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_filters (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    query VARCHAR(100) NOT NULL,
    min_price INTEGER DEFAULT 0,
    max_price INTEGER DEFAULT 0,
    city VARCHAR(50) DEFAULT '',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, name)
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_user_filters_user_id ON user_filters(user_id);