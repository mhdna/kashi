-- Collapses back to one register per branch: each branch keeps its oldest
-- register's key. Rows that only differ by register (shift/attendance refs,
-- client ids reported by a second till) can't be represented any more and
-- are dropped.
ALTER TABLE branch_settings DROP COLUMN oversell_mode;

ALTER TABLE transfers DROP COLUMN register_id;

DELETE FROM branch_sync_conflicts a USING branch_sync_conflicts b
WHERE a.entity = 'client' AND b.entity = 'client' AND a.branch_id = b.branch_id
  AND split_part(a.ref, ':', 2) = split_part(b.ref, ':', 2) AND a.id > b.id;
UPDATE branch_sync_conflicts SET ref = split_part(ref, ':', 2)
WHERE entity = 'client' AND ref LIKE '%:%';

DROP INDEX idx_branch_commands_branch;
DROP INDEX idx_branch_commands_register_pending;
CREATE INDEX idx_branch_commands_branch_pending ON branch_commands (branch_id, status);
ALTER TABLE branch_commands DROP COLUMN register_id;

DELETE FROM client_links a USING client_links b
WHERE a.branch_id = b.branch_id AND a.branch_client_id = b.branch_client_id AND a.register_id > b.register_id;
ALTER TABLE client_links DROP CONSTRAINT client_links_pkey;
ALTER TABLE client_links ADD PRIMARY KEY (branch_id, branch_client_id);
ALTER TABLE client_links DROP COLUMN register_id;

DELETE FROM branch_attendance_changes a USING branch_attendance_changes b
WHERE a.branch_id = b.branch_id AND a.client_ref = b.client_ref AND a.id > b.id;
ALTER TABLE branch_attendance_changes DROP CONSTRAINT branch_attendance_changes_register_id_client_ref_key;
ALTER TABLE branch_attendance_changes ADD CONSTRAINT branch_attendance_changes_branch_id_client_ref_key UNIQUE (branch_id, client_ref);
ALTER TABLE branch_attendance_changes DROP COLUMN register_id;

DELETE FROM branch_shift_receipt_images WHERE branch_shift_id IN (
    SELECT a.id FROM branch_shifts a JOIN branch_shifts b
    ON a.branch_id = b.branch_id AND a.client_ref = b.client_ref AND a.id > b.id);
DELETE FROM branch_shift_upload_tokens WHERE branch_shift_id IN (
    SELECT a.id FROM branch_shifts a JOIN branch_shifts b
    ON a.branch_id = b.branch_id AND a.client_ref = b.client_ref AND a.id > b.id);
DELETE FROM branch_shifts a USING branch_shifts b
WHERE a.branch_id = b.branch_id AND a.client_ref = b.client_ref AND a.id > b.id;
ALTER TABLE branch_shifts DROP CONSTRAINT branch_shifts_register_id_client_ref_key;
ALTER TABLE branch_shifts ADD CONSTRAINT branch_shifts_branch_id_client_ref_key UNIQUE (branch_id, client_ref);
ALTER TABLE branch_shifts DROP COLUMN register_id;

ALTER TABLE branch_invoices DROP COLUMN register_id;

ALTER TABLE branches ADD COLUMN api_key_hash text;
UPDATE branches b SET api_key_hash = r.api_key_hash
FROM (SELECT DISTINCT ON (branch_id) branch_id, api_key_hash FROM branch_registers ORDER BY branch_id, id) r
WHERE r.branch_id = b.id;
UPDATE branches SET api_key_hash = '' WHERE api_key_hash IS NULL;
ALTER TABLE branches ALTER COLUMN api_key_hash SET NOT NULL;

DROP TABLE branch_registers;
