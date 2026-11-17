-- Every client gets a code, in one statement that only locks the rows it
-- writes, so nothing reading clients waits. The code comes from the id
-- scrambled by an affine map mod 23^5 (2654435 isn't a multiple of 23, so
-- the map is one to one): no two ids share one, and neighbours' codes look
-- unrelated. updated_at moves so every till pulls the codes once.
UPDATE clients
SET referral_code = referral_code_from((id * 2654435 + 1171717) % 6436343),
    updated_at = now()
WHERE referral_code IS NULL AND id < 6436343;
