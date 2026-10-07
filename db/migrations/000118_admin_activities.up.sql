-- What people did in kashi itself (the home page's "In kashi" feed): one row
-- per successful change made through the admin API, written by
-- recordAdminActivity (api/admin_activity.go). Branch-side activity (sales,
-- shifts, expenses from the tills) has its own feed and isn't repeated here.
CREATE TABLE admin_activities (
    id bigserial PRIMARY KEY,
    user_id bigint REFERENCES users (id) ON DELETE SET NULL,
    method text NOT NULL,
    -- The gin route pattern, e.g. /transfers/:id/approve
    route text NOT NULL,
    -- The :id in the route, when it has one.
    entity_id bigint,
    -- A short human description, e.g. "approved transfer #12".
    summary text NOT NULL,
    -- A name/type/code picked from the request, or the branch's name.
    detail text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_activities_created_at_idx ON admin_activities (created_at DESC);
