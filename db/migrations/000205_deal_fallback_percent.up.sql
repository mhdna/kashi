-- A "Buy 2 get 1 free" deal can name a fallback discount: the units that
-- don't make up a set get it. One shirt on "Buy 2 get 1, otherwise 30%" is
-- 30% off; so are two. At 3 units the set is the deal (two paid, one free,
-- and the fallback is not added on top); at 4, a set and one unit at 30%.
-- The till does the arithmetic (kashi-pos deals.go) and, for a full set,
-- charges whichever is cheaper for the customer, the deal or the fallback
-- on every unit. 0 is no fallback, which is every deal until now.
ALTER TABLE deals ADD COLUMN fallback_percent smallint NOT NULL DEFAULT 0
    CHECK (fallback_percent >= 0 AND fallback_percent <= 100);
ALTER TABLE deals ADD CONSTRAINT deals_fallback_only_buy_get
    CHECK (fallback_percent = 0 OR kind = 'buy_get');
