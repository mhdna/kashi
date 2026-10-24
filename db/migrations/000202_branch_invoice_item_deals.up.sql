-- A till line sold on a deal (000201, priced by kashi-pos): the deal's name
-- as the till printed it, and what the deal took off the line, in cents.
-- line_total already has it off; these only say why it is lower, so the
-- admin views can mark free units. A returned line carries them signed like
-- its quantity.
ALTER TABLE branch_invoice_items ADD COLUMN deal text NOT NULL DEFAULT '';
ALTER TABLE branch_invoice_items ADD COLUMN deal_discount bigint NOT NULL DEFAULT 0;
