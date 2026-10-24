-- Per-branch mirrors of kashi-pos's discount_toggle and
-- discount_toggle_default_off: salespersons can tap a line's list discount or
-- deal to turn it off for that sale (never change it), and new lines can
-- start with it off. Both off by default, matching the till's behaviour
-- before they existed.
ALTER TABLE branch_settings
  ADD COLUMN discount_toggle boolean NOT NULL DEFAULT false,
  ADD COLUMN discount_toggle_default_off boolean NOT NULL DEFAULT false;
