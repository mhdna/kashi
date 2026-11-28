-- Google Maps, WhatsApp and An ad or flyer go too (till migration 000153):
-- the starting options are Social media, A friend or family and Walking by.
-- One a customer already gave is hidden instead, so their answer stays.
DELETE FROM client_sources s
WHERE s.uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000006', '6f1c2a00-5e1d-4c7a-9b00-000000000007', '6f1c2a00-5e1d-4c7a-9b00-000000000008')
  AND NOT EXISTS (SELECT 1 FROM clients c WHERE c.source_id = s.id);

UPDATE client_sources SET is_active = false, updated_at = now()
WHERE uid IN ('6f1c2a00-5e1d-4c7a-9b00-000000000006', '6f1c2a00-5e1d-4c7a-9b00-000000000007', '6f1c2a00-5e1d-4c7a-9b00-000000000008');
