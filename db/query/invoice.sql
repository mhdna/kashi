-- name: CreateInvoice :one
INSERT INTO invoices (
  cashbox_id,
  shift_id,
  invoice_code,
  invoice_index,
  year,
  client_id,
  inventory_id,
  discount,
  subtotal,
  discounted_total,
  grand_total
) 
VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11 )
RETURNING *;

-- name: AddInvoiceProduct :one
INSERT INTO invoice_products (
  invoice_id,
  product_id,
  unit_price,
  line_total,
  discount,
  quantity
) 
VALUES ( $1, $2, $3, $4, $5, $6 )
RETURNING *;

-- name: GetInvoice :one
SELECT * FROM sales_invoices
WHERE invoice_id = $1 LIMIT 1;

-- name: ListInvoices :many
SELECT *
FROM invoices
ORDER BY created_at
DESC
LIMIT $1
OFFSET $2;

-- name: IncrementInvoicesIndex :one
INSERT INTO invoice_indexes  (year, cashbox_id, type, last_index)
VALUES ($1, $2, $3, 1)
ON CONFLICT (year, cashbox_id)
DO UPDATE SET last_index = sales_invoices_indexes.last_index + 1
RETURNING last_index;

-- name: DecrementInvoicesIndex :one
INSERT INTO invoice_indexes  (year, cashbox_id, type, last_index)
VALUES ($1, $2, $3, 1)
ON CONFLICT (year, cashbox_id)
DO UPDATE SET last_index = sales_invoices_indexes.last_index - 1
RETURNING last_index;