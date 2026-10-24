-- Per-branch mirrors of kashi-pos's product_search_code_only,
-- price_edit_restricted and price_edit_codes: the product list matches codes
-- only and hides names; manual price/discount edits are limited to the listed
-- codes (comma-separated). All off/empty by default, matching the till's
-- behaviour before they existed.
ALTER TABLE branch_settings
  ADD COLUMN product_search_code_only boolean NOT NULL DEFAULT false,
  ADD COLUMN price_edit_restricted boolean NOT NULL DEFAULT false,
  ADD COLUMN price_edit_codes text NOT NULL DEFAULT '';
