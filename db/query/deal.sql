-- name: CreateDeal :one
INSERT INTO deals (name, kind, buy_qty, get_qty, percent, bundle_price)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetDeal :one
SELECT * FROM deals WHERE id = $1;

-- name: ListDeals :many
-- Every deal, with how many list rows and rules put it on products, so the
-- page can say where a deal is used before it is changed or deleted.
SELECT d.*,
       (SELECT COUNT(*) FROM discount_list_items i WHERE i.deal_id = d.id)::bigint AS variants,
       (SELECT COUNT(*) FROM discount_list_rules r WHERE r.deal_id = d.id)::bigint AS rules
FROM deals d
ORDER BY d.name, d.id;

-- name: UpdateDeal :one
UPDATE deals
SET name = $2, kind = $3, buy_qty = $4, get_qty = $5, percent = $6, bundle_price = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDeal :exec
DELETE FROM deals WHERE id = $1;

-- name: ListDealsForDiscountListSync :many
-- The deals a list's rows name: they travel with the list, so the till
-- always has the terms of every deal it can price.
SELECT d.* FROM deals d
WHERE d.id IN (SELECT deal_id FROM discount_list_items WHERE discount_list_id = $1)
ORDER BY d.id;
