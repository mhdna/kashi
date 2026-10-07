-- Roles grant access per page, grouped under the existing sections. A row
-- with page = '' is the section itself: its presence means the section is
-- enabled for the role. A row with a page key is that page's level. Roles
-- saved before this have only section rows; api/access.go lets their pages
-- inherit the section's level, so nobody's access changes until the role is
-- edited.
ALTER TABLE role_permissions ADD COLUMN page text NOT NULL DEFAULT '';
ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_pkey;
ALTER TABLE role_permissions ADD PRIMARY KEY (role_id, section, page);
