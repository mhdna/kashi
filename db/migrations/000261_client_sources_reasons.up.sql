-- The starting "heard about us" options, reworked: SMS and Other go, the
-- friend option takes family too, and Google Maps, WhatsApp and an ad or
-- flyer join. Fixed uids, as in the till's migration 000152. An option a
-- customer already gave is hidden instead of deleted, so their answer stays.
DELETE FROM client_sources s
WHERE s.uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000002', '6f1c2a00-5e1d-4c7a-9b00-000000000005')
  AND NOT EXISTS (SELECT 1 FROM clients c WHERE c.source_id = s.id);

UPDATE client_sources SET is_active = false, updated_at = now()
WHERE uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000002', '6f1c2a00-5e1d-4c7a-9b00-000000000005');

UPDATE client_sources SET name = 'A friend or family', updated_at = now()
WHERE uid = '6f1c2a00-5e1d-4c7a-9b00-000000000003' AND name = 'A friend';

INSERT INTO client_sources (uid, name, sort_order) VALUES
    ('6f1c2a00-5e1d-4c7a-9b00-000000000006', 'Google Maps', 6),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000007', 'WhatsApp', 7),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000008', 'An ad or flyer', 8)
ON CONFLICT (uid) DO NOTHING;
