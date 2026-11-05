-- A till's invoices list shows each sale's WhatsApp poll answer
-- (api/branch_invoice_ratings.go), looked up by branch and invoice code.
CREATE INDEX whatsapp_messages_rated_invoice_idx
  ON whatsapp_messages (branch_id, invoice_code) WHERE poll_rating IS NOT NULL;
