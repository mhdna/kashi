-- A deal's fallback ("Buy 2 get 1 free, otherwise 30%") normally also covers
-- the units a full set leaves over: at 4 units, a set and one at 30%. With
-- this on, the fallback is only for a cart that makes no set at all (1 or 2
-- units on Buy 2 get 1); once a set forms, the leftover units pay full price.
-- Off is every deal until now.
ALTER TABLE deals ADD COLUMN fallback_only_without_set boolean NOT NULL DEFAULT false;
