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

CREATE TABLE saved_listings (
    id SERIAL PRIMARY KEY,
    filter_id INTEGER REFERENCES user_filters(id) ON DELETE CASCADE,
    url VARCHAR(500) UNIQUE NOT NULL,
    title VARCHAR(300),
    price VARCHAR(500),
    location VARCHAR(200),
    is_notified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_user_filters_user_id ON user_filters(user_id);
CREATE INDEX idx_saved_listings_filter_id ON saved_listings(filter_id);
CREATE INDEX idx_saved_listings_url ON saved_listings(url);
CREATE INDEX idx_saved_listings_is_notified ON saved_listings(is_notified);