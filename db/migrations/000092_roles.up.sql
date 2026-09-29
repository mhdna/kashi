-- Roles for admin-site users. A role is a named set of per-section access
-- levels (see api/access.go for the section list); each user has exactly one.
-- An is_admin role bypasses the per-section rows and is the only kind that can
-- manage users and roles, so it never needs any rows in role_permissions.
CREATE TABLE roles (
    id         bigserial PRIMARY KEY,
    name       text NOT NULL UNIQUE,
    is_admin   boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- One row per (role, section). can_edit means add/edit/delete and is only
-- meaningful together with can_view -- the API and UI both treat edit as
-- implying view.
CREATE TABLE role_permissions (
    role_id  bigint NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    section  text NOT NULL,
    can_view boolean NOT NULL DEFAULT false,
    can_edit boolean NOT NULL DEFAULT false,
    PRIMARY KEY (role_id, section)
);

INSERT INTO roles (name, is_admin) VALUES ('Admin', true);

-- Everyone who can log in today keeps full access; restrict them afterwards
-- from Settings > Roles.
ALTER TABLE users ADD COLUMN role_id bigint REFERENCES roles (id);
UPDATE users SET role_id = (SELECT id FROM roles WHERE is_admin ORDER BY id LIMIT 1);
ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;
