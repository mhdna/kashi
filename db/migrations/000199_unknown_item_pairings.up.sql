-- A phone paired with one till: it scanned that till's QR code once and from
-- then on shows whatever unknown items the till is ringing up, sale after
-- sale. The till keeps its cart's session here as it changes, so a unit is
-- photographed the moment it is added instead of at the payment.
CREATE TABLE unknown_item_pairings (
  branch_id bigint NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
  register_id bigint NOT NULL,
  token text NOT NULL UNIQUE,
  -- The session of the cart on the till right now; NULL between carts.
  current_session_id bigint REFERENCES unknown_item_sessions (id) ON DELETE SET NULL,
  -- The last time the paired phone asked for work: the till shows it online.
  phone_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (branch_id, register_id)
);

-- A session is opened while the cart is still being rung up; it only belongs
-- on the Waiting tab once the cashier is at the payment.
ALTER TABLE unknown_item_sessions ADD COLUMN checkout_at timestamptz;
UPDATE unknown_item_sessions SET checkout_at = created_at;
