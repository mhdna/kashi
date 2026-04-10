-- name: CreateSalesInvoice :one
INSERT INTO sales_invoices (
  cashbox_id,
  invoice_number,
  inventory_id,
  client_id,
  amount,
  net_amount,
  discount,
  currency_code
) 
VALUES ( $1, $2, $3, $4, $5, $6, $7, $8 )
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

-- name: CountSalesInvoicesThisYear :one
SELECT count(*) FROM sales_invoices
WHERE cashbox_id = $1
AND created_at >= date_trunc('year', now() AT TIME ZONE 'UTC');

-- name: CreateReturnInvoice :one
INSERT INTO return_invoices (
  invoice_number,
  sales_invoice_id
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: GetReturnInvoice :one
SELECT * FROM return_invoices
WHERE id = $1 LIMIT 1;

-- name: CountReturnInvoicesThisYear :one
SELECT count(*) FROM return_invoices
WHERE sales_invoice_id = $1
AND created_at >= date_trunc('year', now() AT TIME ZONE 'UTC');
