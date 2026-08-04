ALTER TABLE stripe_plans
ADD CONSTRAINT stripe_price_id_unique UNIQUE (stripe_price_id);
