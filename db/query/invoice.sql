-- name: CreateSalesInvoice :one
INSERT INTO sales_invoices (
  invoice_number,
  inventory_id,
  client_id,
  amount,
  discount,
  net_amount
) 
VALUES ( $1, $2, $3, $4, $5, $6 )
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

-- name: CreateReturnInvoice :one
INSERT INTO return_invoices (
  invoice_number,
  sales_invoice_id
) 
VALUES ( $1, $2 )
RETURNING *;