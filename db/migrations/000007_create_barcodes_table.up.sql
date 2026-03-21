create table if not exists "barcodes" (
  barcode bigint not null unique,
  product_id bigint not null references products(id),
  color_id bigint references colors(id),
  size_id bigint references sizes(id),
  version int NOT NULL DEFAULT 1,
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
