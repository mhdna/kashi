-- New barcodes number from barcode_item_seq, which sat at 82 while imported products
-- held nearly every number up to about 10,000, in runs of up to 944. Nearly every new
-- product hit a taken barcode. Move it past the block they fill. Numbers above a
-- million are a few scattered ones, far off and handled by retrying.
SELECT setval('barcode_item_seq', GREATEST(
  (SELECT last_value FROM barcode_item_seq),
  COALESCE((SELECT max(substring(barcode, 4, 9)::bigint) FROM product_variants
            WHERE barcode ~ '^800[0-9]{10}$' AND substring(barcode, 4, 9)::bigint < 1000000), 0) + 1
));
