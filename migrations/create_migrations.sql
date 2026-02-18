CREATE TABLE "users" (
  "id" int PRIMARY KEY,
  "password" varchar,
  "username" varchar,
  "role" varchar,
  "created_at" timestamp
);

CREATE TABLE "employees" (
  "id" int PRIMARY KEY,
  "first_name" varchar,
  "last_name" varchar,
  "position" varchar,
  "gender" varchar,
  "joined_at" timestamp,
  "branch_id" int
);

CREATE TABLE "attendences" (
  "date" datetime PRIMARY KEY,
  "employee_id" int,
  "check_in_time" timestamp,
  "check_out_time" timestamp,
  "location_id" int,
  "picture_url" varchar
);

CREATE TABLE "locations" (
  "id" int PRIMARY KEY,
  "name" varchar,
  "code" varchar UNIQUE,
  "longitude" varchar,
  "latitude" varchar
);

CREATE TABLE "products" (
  "id" int PRIMARY KEY,
  "name" varchar NOT NULL,
  "kind" varchar NOT NULL,
  "category_id" int,
  "sub_category_id" int,
  "unit" varchar,
  "type" varchar,
  "year" varchar,
  "season" varchar,
  "brand_id" int,
  "price" int
);

CREATE TABLE "product_varaints" (
  "product_id" int,
  "size" int,
  "color" int,
  "barcode" int PRIMARY KEY
);

CREATE TABLE "colors" (
  "id" int PRIMARY KEY,
  "name" varchar UNIQUE,
  "value" varchar
);

CREATE TABLE "sizes" (
  "id" int PRIMARY KEY,
  "name" varchar UNIQUE,
  "unit" varchar NOT NULL,
  "value" int NOT NULL
);

CREATE TABLE "brands" (
  "id" int PRIMARY KEY,
  "name" varchar UNIQUE
);

CREATE TABLE "suppliers" (
  "id" int PRIMARY KEY,
  "name" varchar UNIQUE,
  "country" varchar UNIQUE
);

CREATE TABLE "products_suppliers" (
  "product_id" int,
  "supplier_id" int,
  "prodcut_cost" int
);

CREATE TABLE "products_categories" (
  "id" int PRIMARY KEY,
  "name" varchar
);

CREATE TABLE "products_subcategories" (
  "id" int PRIMARY KEY,
  "parent_category_id" int,
  "name" varchar
);

CREATE TABLE "assets" (
  "id" int PRIMARY KEY,
  "count" int,
  "name" varchar
);

CREATE TABLE "products_location_log" (
  "log_id" int PRIMARY KEY,
  "product_id" int,
  "location_id" int,
  "transfered_in" timestamp,
  "count" int
);

CREATE TABLE "assets_location_log" (
  "log_id" int PRIMARY KEY,
  "asset_id" int,
  "location_id" int,
  "transfered_in" timestamp,
  "count" int
);

CREATE TABLE "customers" (
  "id" int PRIMARY KEY,
  "phone" int UNIQUE,
  "first_name" varchar,
  "last_name" varchar,
  "loyalty_points" int,
  "total_money_spent" int
);

CREATE TABLE "sales_invoices" (
  "id" int PRIMARY KEY,
  "location" int,
  "customer_id" int,
  "salesman_id" int,
  "date_time" timestamp
);

CREATE TABLE "sales_invoice_products" (
  "product_id" int,
  "invoice_id" int,
  "product_discount" int
);

CREATE TABLE "return_invoice" (
  "id" int PRIMARY KEY,
  "location" int,
  "sales_invoice_id" int,
  "date_time" timestamp
);

CREATE TABLE "payment" (
  "currency_id" int,
  "amount" int
);

CREATE TABLE "currencies" (
  "id" int PRIMARY KEY,
  "name" varchar
);

ALTER TABLE "employees" ADD FOREIGN KEY ("branch_id") REFERENCES "locations" ("id");

ALTER TABLE "attendences" ADD FOREIGN KEY ("location_id") REFERENCES "locations" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("category_id") REFERENCES "products_categories" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("sub_category_id") REFERENCES "products_subcategories" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("brand_id") REFERENCES "brands" ("id");

ALTER TABLE "product_varaints" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "product_varaints" ADD FOREIGN KEY ("size") REFERENCES "sizes" ("id");

ALTER TABLE "product_varaints" ADD FOREIGN KEY ("color") REFERENCES "colors" ("id");

ALTER TABLE "products_suppliers" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "products_suppliers" ADD FOREIGN KEY ("supplier_id") REFERENCES "suppliers" ("id");

ALTER TABLE "products_subcategories" ADD FOREIGN KEY ("parent_category_id") REFERENCES "products_categories" ("id");

ALTER TABLE "products_location_log" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "products_location_log" ADD FOREIGN KEY ("location_id") REFERENCES "locations" ("id");

ALTER TABLE "assets_location_log" ADD FOREIGN KEY ("asset_id") REFERENCES "assets" ("id");

ALTER TABLE "assets_location_log" ADD FOREIGN KEY ("location_id") REFERENCES "locations" ("id");

ALTER TABLE "sales_invoices" ADD FOREIGN KEY ("location") REFERENCES "locations" ("id");

ALTER TABLE "sales_invoices" ADD FOREIGN KEY ("customer_id") REFERENCES "customers" ("id");

ALTER TABLE "sales_invoices" ADD FOREIGN KEY ("salesman_id") REFERENCES "employees" ("id");

ALTER TABLE "sales_invoice_products" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "sales_invoice_products" ADD FOREIGN KEY ("invoice_id") REFERENCES "sales_invoices" ("id");

ALTER TABLE "return_invoice" ADD FOREIGN KEY ("location") REFERENCES "locations" ("id");

ALTER TABLE "return_invoice" ADD FOREIGN KEY ("sales_invoice_id") REFERENCES "sales_invoices" ("id");

ALTER TABLE "payment" ADD FOREIGN KEY ("currency_id") REFERENCES "currencies" ("id");
