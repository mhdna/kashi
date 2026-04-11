-- name: CreateSalesInvoice :one
INSERT INTO sales_invoices (
  cashbox_id,
  invoice_index,
  invoice_code,
  inventory_id,
  year,
  client_id,
  amount,
  net_amount,
  discount,
  currency_code
) 
VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10 )
RETURNING *;

-- name: AddSalesInvoiceProduct :one
INSERT INTO sales_invoice_products (
  invoice_id,
  product_id,
  quantity
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetSalesInvoice :one
SELECT * FROM sales_invoices
WHERE id = $1 LIMIT 1;

-- name: ListSalesInvoices :many
SELECT * FROM sales_invoices
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: NextSalesInvoiceIndexIncrement :one
INSERT INTO sales_invoices_indexes  (year, cashbox_id, last_index)
VALUES ($1, $2, 1)
ON CONFLICT (year, cashbox_id)
DO UPDATE SET last_index = sales_invoices_indexes.last_index + 1
RETURNING last_index;

-- name: CreateReturnInvoice :one
INSERT INTO return_invoices (
  invoice_index,
  invoice_code,
  cashbox_id,
  sales_invoice_id
) 
VALUES ( $1, $2, $3, $4 )
RETURNING *;

-- name: GetReturnInvoice :one
SELECT * FROM return_invoices
WHERE id = $1 LIMIT 1;

-- name: NextReturnInvoiceIndexIncrement :one
INSERT INTO return_invoices_indexes (year, cashbox_id, last_index)
VALUES ($1, $2, 1)
ON CONFLICT (year, cashbox_id)
DO UPDATE SET last_index = return_invoices_indexes.last_index + 1
RETURNING last_index;
