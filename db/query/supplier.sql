-- name: CreateSupplier :one
INSERT INTO suppliers (
  name,
  phone,
  country,
  address,
  address_latitude,
  address_longitude
) 
VALUES ( $1, $2, $3, $4, $5, $6 )
RETURNING *;

-- name: GetSupplier :one
SELECT * FROM suppliers
WHERE id = $1 LIMIT 1;

-- name: ListSuppliers :many
SELECT * FROM suppliers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: AddSupplierProduct :one
INSERT INTO product_suppliers (
  product_id,
  supplier_id
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: AddSupplierProductCost :one
INSERT INTO product_supplier_costs (
  product_supplier_id,
  unit_cost,
  currency_id
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: DeleteSupplierProduct :exec
delete from product_suppliers
where id = $1;