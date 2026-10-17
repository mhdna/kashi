-- What was actually sent, for the Messages tab's chat view: the message, the
-- poll as asked and the follow-up. Empty for messages sent before this, which
-- the view rebuilds from the sale's fields and today's wording.
ALTER TABLE whatsapp_messages ADD COLUMN body text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN poll_question text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN poll_options text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN followup_body text NOT NULL DEFAULT '';
