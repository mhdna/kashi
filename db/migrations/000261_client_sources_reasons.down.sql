DELETE FROM client_sources s
WHERE s.uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000006', '6f1c2a00-5e1d-4c7a-9b00-000000000007', '6f1c2a00-5e1d-4c7a-9b00-000000000008')
  AND NOT EXISTS (SELECT 1 FROM clients c WHERE c.source_id = s.id);

UPDATE client_sources SET name = 'A friend' WHERE uid = '6f1c2a00-5e1d-4c7a-9b00-000000000003' AND name = 'A friend or family';
UPDATE client_sources SET is_active = true WHERE uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000002', '6f1c2a00-5e1d-4c7a-9b00-000000000005');
INSERT INTO client_sources (uid, name, sort_order) VALUES
    ('6f1c2a00-5e1d-4c7a-9b00-000000000002', 'SMS', 2),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000005', 'Other', 5)
ON CONFLICT (uid) DO NOTHING;
