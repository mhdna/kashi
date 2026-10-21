-- With it on, the follow-up after the rating poll goes to a client only once:
-- one who got it on an earlier visit isn't thanked again (api/whatsapp_poll.go).
ALTER TABLE app_settings ADD COLUMN whatsapp_followup_first_only boolean NOT NULL DEFAULT false;
