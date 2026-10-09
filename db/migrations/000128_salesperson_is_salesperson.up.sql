-- Not everyone who punches the attendance device sells: a branch's roster is
-- its staff, and is_salesperson marks the ones a till offers at checkout.
-- Everyone already on a roster was added to sell, so they stay salespersons.
ALTER TABLE salespersons ADD COLUMN is_salesperson boolean NOT NULL DEFAULT true;
