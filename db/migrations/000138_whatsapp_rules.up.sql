-- Rules deciding which post-sale WhatsApp messages go out, and a log of
-- every message kashi decided on (sent, skipped by a rule, or failed) for
-- the WhatsApp page's stats. See api/whatsapp.go and api/whatsapp_rules.go.

-- Ordered by position; the first enabled rule whose conditions all match
-- decides. action 'skip' sends nothing; 'send' sends, with this rule's
-- template when template_name is set, else the settings' default. A sale
-- no rule matches gets the default template.
CREATE TABLE whatsapp_rules (
    id bigserial PRIMARY KEY,
    position integer NOT NULL DEFAULT 0,
    name text NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    action text NOT NULL DEFAULT 'send' CHECK (action IN ('send', 'skip')),
    -- whatsAppRuleConditions in api/whatsapp_rules.go; {} matches every sale.
    conditions jsonb NOT NULL DEFAULT '{}',
    template_name text NOT NULL DEFAULT '',
    template_lang text NOT NULL DEFAULT '',
    template_params text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- rule_id has no foreign key: rules are replaced as a whole list on save,
-- and the log keeps rule_name so old rows still read right.
CREATE TABLE whatsapp_messages (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    branch_id bigint REFERENCES branches(id) ON DELETE SET NULL,
    invoice_code text NOT NULL DEFAULT '',
    client_name text NOT NULL DEFAULT '',
    -- Digits only, as sent to the Cloud API.
    client_phone text NOT NULL DEFAULT '',
    grand_total_cents bigint NOT NULL DEFAULT 0,
    item_count integer NOT NULL DEFAULT 0,
    rule_id bigint,
    rule_name text NOT NULL DEFAULT '',
    template_name text NOT NULL DEFAULT '',
    status text NOT NULL CHECK (status IN ('sent', 'skipped', 'failed')),
    reason text NOT NULL DEFAULT '',
    -- Meta's message id (wamid.…) for a sent message.
    wamid text NOT NULL DEFAULT '',
    -- What this message cost, in millionths of a USD: the price per message
    -- at the time it was sent, so a later price change doesn't rewrite
    -- history. 0 for skipped/failed.
    cost_micros bigint NOT NULL DEFAULT 0
);
CREATE INDEX whatsapp_messages_created_at_idx ON whatsapp_messages (created_at);
CREATE INDEX whatsapp_messages_client_phone_idx ON whatsapp_messages (client_phone, created_at);

-- Price Meta charges per delivered template message (USD millionths), set
-- on the WhatsApp page; Meta's rate depends on template category and the
-- recipient's country, so this is the business's own figure.
ALTER TABLE app_settings ADD COLUMN whatsapp_message_cost_micros bigint NOT NULL DEFAULT 0;
