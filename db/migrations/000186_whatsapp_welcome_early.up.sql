-- Settings > WhatsApp: a new client's welcome (the channel-first message) goes
-- out as soon as the till has their name and number, not after checkout. The
-- checkout then finds it (a whatsapp_messages row with no invoice yet) and
-- sends nothing more; a returning client still gets their receipt message.
ALTER TABLE app_settings ADD COLUMN whatsapp_welcome_early boolean NOT NULL DEFAULT false;
