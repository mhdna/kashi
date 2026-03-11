create table if not exists "barcodes" (
  "barcode" bigint not null unique,
  "product_id" bigint not null references "products",
  "color_id" bigint references "colors",
  "size_id" bigint references "sizes",
  "version" int NOT NULL DEFAULT 1
);
