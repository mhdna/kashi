-- The bridge reports a message's ticks (sent, delivered, read) back by this
-- ref; empty for Meta's messages and ones sent before ticks were tracked.
ALTER TABLE whatsapp_messages ADD COLUMN bridge_ref text NOT NULL DEFAULT '';
CREATE INDEX whatsapp_messages_bridge_ref_idx ON whatsapp_messages (bridge_ref) WHERE bridge_ref <> '';
