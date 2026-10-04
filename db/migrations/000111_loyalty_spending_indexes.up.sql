CREATE INDEX IF NOT EXISTS idx_invoices_client_id ON invoices (client_id);
CREATE INDEX IF NOT EXISTS idx_branch_invoices_register_branch_client ON branch_invoices (register_id, branch_client_id);
