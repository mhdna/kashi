-- Singleton row holding the admin's saved "default settings" template --
-- the operational subset of branch_settings (pricing/exchange, behaviour,
-- page access, printing preferences, attendance behaviour) that can be
-- pushed to every branch at once via "Apply to all". Branch identity
-- (name, storefront details), per-device fields (printer/COM IDs, Akuvox
-- IP/username) and secrets are deliberately not part of this template --
-- those stay branch-specific and are edited per branch only.
CREATE TABLE default_branch_settings (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
