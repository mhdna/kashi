-- "Unknown item" sales: a till sells a product it has no code for after taking
-- a picture of each unit on a phone. A session is one checkout's set of
-- pictures; a slot is one unit that needs a picture (two pictures can share
-- one image when the units are the same item). Once the invoice lands, an
-- admin matches each slot to a real product and the sold line is relabelled.
CREATE TABLE unknown_item_sessions (
  id bigserial PRIMARY KEY,
  branch_id bigint NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
  register_id bigint NOT NULL,
  client_ref text NOT NULL,
  token text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  branch_invoice_id bigint REFERENCES branch_invoices (id) ON DELETE SET NULL,
  -- The cashier asked a manager to let the sale through without pictures.
  help_requested_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (branch_id, client_ref)
);

CREATE TABLE unknown_item_images (
  id bigserial PRIMARY KEY,
  session_id bigint NOT NULL REFERENCES unknown_item_sessions (id) ON DELETE CASCADE,
  file_path text NOT NULL,
  content_type text NOT NULL,
  size_bytes bigint NOT NULL,
  rotation smallint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE unknown_item_slots (
  id bigserial PRIMARY KEY,
  session_id bigint NOT NULL REFERENCES unknown_item_sessions (id) ON DELETE CASCADE,
  slot_ref text NOT NULL,
  position int NOT NULL,
  label text NOT NULL DEFAULT '',
  unit_price bigint NOT NULL,
  discount smallint NOT NULL DEFAULT 0,
  image_id bigint REFERENCES unknown_item_images (id) ON DELETE SET NULL,
  resolved_variant_id bigint REFERENCES product_variants (id) ON DELETE SET NULL,
  resolved_at timestamptz,
  -- A manager let this unit through with no picture (the cashier may have sent
  -- it privately); it still has to be resolved like any other.
  waived_at timestamptz,
  waived_by text NOT NULL DEFAULT '',
  UNIQUE (session_id, slot_ref)
);

CREATE INDEX ON unknown_item_slots (session_id);

-- A sold line the till could not name: set on lines whose barcode is the
-- till's UNKNOWN placeholder, cleared when the line is resolved.
ALTER TABLE branch_invoice_items ADD COLUMN unknown boolean NOT NULL DEFAULT false;
