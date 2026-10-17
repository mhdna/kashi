-- A reply's media (a voice note, photo...) kept by the bridge: its type, so
-- the chat view can play or show it from GET /media on the bridge. Empty for
-- a text reply, or media the bridge didn't keep.
ALTER TABLE whatsapp_poll_replies ADD COLUMN media_type text NOT NULL DEFAULT '';
