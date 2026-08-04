CREATE TABLE IF NOT EXISTS stripe_plans (
    id SERIAL PRIMARY KEY,
    stripe_product_id TEXT NOT NULL,
    stripe_price_id TEXT NOT NULL,
    name TEXT,
    description TEXT,
    active BOOL NOT NULL,
    unit_amount INT,
    currency TEXT,
    interval TEXT,
    interval_count INT,
    trial_days INT, 
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
