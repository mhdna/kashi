DELETE FROM role_permissions WHERE page <> '';
ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_pkey;
ALTER TABLE role_permissions DROP COLUMN page;
ALTER TABLE role_permissions ADD PRIMARY KEY (role_id, section);
