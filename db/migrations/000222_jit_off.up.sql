-- JIT compiling a query took 2-3s on the 1-CPU server, far longer than the
-- queries it was meant to speed up: opening a discount list spent most of its
-- 6s there. Off for the app's own role, so it applies from the next connection.
ALTER ROLE CURRENT_USER SET jit = off;
