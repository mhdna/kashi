-- A code's characters: none that read like another (no 0/O, 1/I/L, 2/Z,
-- 5/S, 8/B, U/V). 23 of them, a prime, so the check character (positions
-- weighted 1, 2, 3 ..., mod 23) catches every single wrong character and
-- every swap of two neighbours. The tills check codes the same way.
CREATE OR REPLACE FUNCTION referral_check_char(body text) RETURNS text
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    alphabet constant text := 'ACDEFGHJKMNPQRTWXY34679';
    s int := 0;
    v int;
BEGIN
    FOR i IN 1..length(body) LOOP
        v := strpos(alphabet, substr(body, i, 1)) - 1;
        IF v < 0 THEN
            RETURN NULL;
        END IF;
        s := s + i * v;
    END LOOP;
    RETURN substr(alphabet, (s % 23) + 1, 1);
END $$;

-- The code made from a number below 23^5: its five base-23 digits and the
-- check character. Two numbers never give the same code.
CREATE OR REPLACE FUNCTION referral_code_from(n bigint) RETURNS text
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    alphabet constant text := 'ACDEFGHJKMNPQRTWXY34679';
    body text := '';
    rest bigint := n;
BEGIN
    FOR i IN 1..5 LOOP
        body := substr(alphabet, (rest % 23)::int + 1, 1) || body;
        rest := rest / 23;
    END LOOP;
    RETURN body || referral_check_char(body);
END $$;

-- A random unused code, for clients made from now on.
CREATE OR REPLACE FUNCTION new_referral_code() RETURNS text
LANGUAGE plpgsql VOLATILE AS $$
DECLARE
    code text;
BEGIN
    LOOP
        code := referral_code_from(floor(random() * 6436343)::bigint);
        IF NOT EXISTS (SELECT 1 FROM clients WHERE referral_code = code) THEN
            RETURN code;
        END IF;
    END LOOP;
END $$;

-- No default yet: adding the column is instant, and every client still
-- without one gets it in 247.
ALTER TABLE clients ADD COLUMN referral_code text;
