-- The post-sale messages link the shop's WhatsApp channel instead of its
-- Instagram page: {channel} replaces {instagram}. An Instagram address isn't
-- a channel link, so it isn't kept, and a saved message's {instagram} line
-- goes (it would otherwise be sent as typed). So does the old default's
-- "Please keep this message" line, dropped from the defaults too.
ALTER TABLE app_settings RENAME COLUMN whatsapp_instagram_url TO whatsapp_channel_url;
UPDATE app_settings SET
    whatsapp_channel_url = '',
    whatsapp_bridge_message = regexp_replace(regexp_replace(whatsapp_bridge_message,
        '\n*[^\n]*\{instagram\}[^\n]*', '', 'g'),
        '\n*Please keep this message: [^\n]*', '', 'g'),
    whatsapp_bridge_link_message = regexp_replace(regexp_replace(whatsapp_bridge_link_message,
        '\n*[^\n]*\{instagram\}[^\n]*', '', 'g'),
        '\n*Please keep this message: [^\n]*', '', 'g');
