-- A star-rating WhatsApp poll after the post-sale message, from our own
-- numbers (api/whatsapp_poll.go). The bridge reads each vote and posts it
-- back to kashi with the poll's ref.

-- Whether to send the poll, its question and its options, one per line,
-- best first (5 lines: the first is 5 stars, the last 1). Empty = defaults.
ALTER TABLE app_settings ADD COLUMN whatsapp_poll_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE app_settings ADD COLUMN whatsapp_poll_question text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_poll_options text NOT NULL DEFAULT '';

-- The poll a message was followed by, and the client's answer: its stars
-- (1-5, NULL until they vote or after they take it back) and the option's
-- text as they saw it.
ALTER TABLE whatsapp_messages ADD COLUMN poll_ref text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN poll_rating smallint CHECK (poll_rating BETWEEN 1 AND 5);
ALTER TABLE whatsapp_messages ADD COLUMN poll_answer text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN poll_voted_at timestamptz;
CREATE UNIQUE INDEX whatsapp_messages_poll_ref_idx ON whatsapp_messages (poll_ref) WHERE poll_ref <> '';
