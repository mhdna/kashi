-- Per-branch mirror of kashi-pos's deal_toggle: salespersons can tap a
-- line's deal (B2G1) to turn it off for that sale, apart from
-- discount_toggle, which now covers only percent discounts. Off by default.
ALTER TABLE branch_settings
  ADD COLUMN deal_toggle boolean NOT NULL DEFAULT false;
