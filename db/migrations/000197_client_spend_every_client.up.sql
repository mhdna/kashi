-- Every client has a client_spend row (zeros until the next refresh fills in
-- the real figures), so the clients list can inner-join it and read the top
-- of an index instead of sorting everyone.
INSERT INTO client_spend (client_id) SELECT id FROM clients ON CONFLICT DO NOTHING;

CREATE FUNCTION client_spend_for_new_client() RETURNS trigger AS $$
BEGIN
  INSERT INTO client_spend (client_id) VALUES (NEW.id) ON CONFLICT DO NOTHING;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER clients_client_spend AFTER INSERT ON clients
  FOR EACH ROW EXECUTE FUNCTION client_spend_for_new_client();
