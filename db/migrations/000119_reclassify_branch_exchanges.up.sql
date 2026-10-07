-- kashi-pos posts exchanges to /branch/return_invoices, so they were all
-- stored as kind 'return'. An exchange is the one with replacement lines
-- (positive quantity); api/branch_sync.go now classifies new ones that way.
UPDATE branch_invoices bi
SET kind = 'exchange'
WHERE bi.kind = 'return'
  AND EXISTS (
    SELECT 1 FROM branch_invoice_items bii
    WHERE bii.branch_invoice_id = bi.id AND bii.quantity > 0
  );
