ALTER TABLE deals DROP CONSTRAINT IF EXISTS deals_fallback_only_buy_get;
ALTER TABLE deals DROP COLUMN IF EXISTS fallback_percent;
