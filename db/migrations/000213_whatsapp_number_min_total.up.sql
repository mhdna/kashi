-- The smallest invoice a number sends for (cents, 0 = every invoice). A
-- smaller sale goes out from the next number that takes it: the branch's
-- own number first, then the default, then the oldest (WhatsAppNumberForBranch).
-- Lets a number that was restricted carry only the sales least likely to be
-- reported, while the rest go from another number.
ALTER TABLE whatsapp_numbers ADD COLUMN min_total_cents bigint NOT NULL DEFAULT 0;
