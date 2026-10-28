-- Invoice discounts: a discount on a whole sale once it reaches a total.
-- "Invoices of $150 or more get 15% off", "$10 off at $100".
--
-- The till checks the total after item discounts and deals, and takes the
-- discount off that, on top of them. When several apply, the one that saves
-- the customer most wins; they never add up. The till spreads the discount
-- over the lines in proportion to what each costs, so a return refunds what
-- the line was actually paid.
--
-- A rule is never deleted, only marked so (deleted_at): the tills learn of
-- a rule going away by its row coming down as not applying to them.
CREATE TABLE invoice_discounts (
    id bigserial PRIMARY KEY,
    -- The rule's identity on the tills.
    uid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name text NOT NULL,
    -- Cents: the sale's total (after item discounts and deals, before tax)
    -- has to be at least this.
    min_total bigint NOT NULL CHECK (min_total >= 0),
    kind text NOT NULL CHECK (kind IN ('percent', 'amount')),
    percent smallint NOT NULL DEFAULT 0 CHECK (percent >= 0 AND percent <= 100),
    -- Cents, for amount.
    amount bigint NOT NULL DEFAULT 0 CHECK (amount >= 0),
    -- Every branch, or only those in invoice_discount_branches.
    all_branches boolean NOT NULL DEFAULT true,
    -- Either end open when null.
    valid_from timestamptz,
    valid_to timestamptz,
    is_active boolean NOT NULL DEFAULT true,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (CASE kind
        WHEN 'percent' THEN percent >= 1 AND amount = 0
        WHEN 'amount' THEN amount > 0 AND percent = 0
    END)
);

CREATE TABLE invoice_discount_branches (
    invoice_discount_id bigint NOT NULL REFERENCES invoice_discounts(id) ON DELETE CASCADE,
    branch_id bigint NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    PRIMARY KEY (invoice_discount_id, branch_id)
);

CREATE INDEX invoice_discounts_updated_at_idx ON invoice_discounts (updated_at, id);

-- What a synced sale's invoice discount was: the rule's name, and the cents
-- it took off the sale (and off each line, its share). 0 / '' for a sale
-- without one, and for tills that predate this.
ALTER TABLE branch_invoices
    ADD COLUMN invoice_discount bigint NOT NULL DEFAULT 0,
    ADD COLUMN invoice_discount_name text NOT NULL DEFAULT '';
ALTER TABLE branch_invoice_items
    ADD COLUMN invoice_discount bigint NOT NULL DEFAULT 0;

-- Tills re-fetch the rules the moment one changes (000109).
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON invoice_discounts
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('invoice_discounts');
