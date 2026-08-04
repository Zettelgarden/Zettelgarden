ALTER TABLE users ADD COLUMN stripe_customer_id TEXT;
ALTER TABLE users ADD COLUMN stripe_subscription_id TEXT;
ALTER TABLE users ADD COLUMN stripe_subscription_status TEXT;
ALTER TABLE users ADD COLUMN stripe_subscription_frequency TEXT;
ALTER TABLE users ADD COLUMN stripe_current_plan TEXT;
