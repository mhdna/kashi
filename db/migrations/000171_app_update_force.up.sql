-- App Updates: a target set with "force" installs as soon as the till has it,
-- even mid-shift (after a short countdown), instead of waiting for the
-- cashier or a closed shift.
ALTER TABLE branch_registers ADD COLUMN target_force boolean NOT NULL DEFAULT false;
