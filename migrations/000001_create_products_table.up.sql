CREATE TABLE IF NOT EXISTS "products" (
  "id" bigserial PRIMARY KEY,
  "code" text not null,
  "name" text NOT NULL,
  "description" text not null,
  "kind" text NOT NULL,
  "category_id" int,
  "is_active" bool not null DEFAULT true,
  "sub_category_id" int,
  "unit" text NOT NULL,
  "type" text NOT NULL,
  "year" int NOT NULL,
  "season" text NOT NULL,
  "brand_id" int,
  "origin_id" int ,
  "price" int NOT NULL,
  "version" int NOT NULL DEFAULT 1,
  "created_at" timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
