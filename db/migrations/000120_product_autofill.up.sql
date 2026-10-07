-- Smart auto-fill on the New Product dialog: rules that read a product's
-- code and name and fill its attributes and price. Evaluated in the UI
-- (ui/src/utils/productAutofill.js); edited on the Auto-fill Rules page.
ALTER TABLE app_settings
  ADD COLUMN product_autofill_enabled BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN product_autofill_rules JSONB NOT NULL DEFAULT '[]';

-- Starting rules. Code like L6KBSWT25: 1st letter is the origin, 2nd digit
-- the year, the trailing number the price. Names fill category, sub-category
-- and type from common words ("Kids boy sweatshirt").
UPDATE app_settings SET product_autofill_rules = '[
  {"source": "code", "match": "char", "position": 1, "pattern": "L", "field": "origin", "value": "Lebanon"},
  {"source": "code", "match": "char", "position": 1, "pattern": "T", "field": "origin", "value": "Turkey"},
  {"source": "code", "match": "char", "position": 2, "pattern": "5", "field": "year", "value": "2025"},
  {"source": "code", "match": "char", "position": 2, "pattern": "6", "field": "year", "value": "2026"},
  {"source": "code", "match": "char", "position": 2, "pattern": "7", "field": "year", "value": "2027"},
  {"source": "code", "match": "char", "position": 2, "pattern": "8", "field": "year", "value": "2028"},
  {"source": "code", "match": "char", "position": 2, "pattern": "9", "field": "year", "value": "2029"},
  {"source": "code", "match": "trailing_number", "position": 1, "pattern": "", "field": "price", "value": ""},
  {"source": "name", "match": "word", "position": 1, "pattern": "kids|kid|children", "field": "category", "value": "Kids"},
  {"source": "name", "match": "word", "position": 1, "pattern": "baby|babies|infant", "field": "category", "value": "Baby"},
  {"source": "name", "match": "word", "position": 1, "pattern": "women|woman|ladies|lady", "field": "category", "value": "Women"},
  {"source": "name", "match": "word", "position": 1, "pattern": "men|man", "field": "category", "value": "Men"},
  {"source": "name", "match": "word", "position": 1, "pattern": "boy|boys", "field": "subcategory", "value": "Boys"},
  {"source": "name", "match": "word", "position": 1, "pattern": "girl|girls", "field": "subcategory", "value": "Girls"},
  {"source": "name", "match": "word", "position": 1, "pattern": "sweatshirt|sweatshirts", "field": "type", "value": "Sweatshirt"},
  {"source": "name", "match": "word", "position": 1, "pattern": "hoodie|hoodies", "field": "type", "value": "Hoodie"},
  {"source": "name", "match": "word", "position": 1, "pattern": "t-shirt|tshirt|tee", "field": "type", "value": "T-Shirt"},
  {"source": "name", "match": "word", "position": 1, "pattern": "shirt|shirts", "field": "type", "value": "Shirt"},
  {"source": "name", "match": "word", "position": 1, "pattern": "sweater|pullover", "field": "type", "value": "Sweater"},
  {"source": "name", "match": "word", "position": 1, "pattern": "jacket|jackets", "field": "type", "value": "Jacket"},
  {"source": "name", "match": "word", "position": 1, "pattern": "coat|coats", "field": "type", "value": "Coat"},
  {"source": "name", "match": "word", "position": 1, "pattern": "jeans|jean", "field": "type", "value": "Jeans"},
  {"source": "name", "match": "word", "position": 1, "pattern": "pants|pant|trousers", "field": "type", "value": "Pants"},
  {"source": "name", "match": "word", "position": 1, "pattern": "shorts|short", "field": "type", "value": "Shorts"},
  {"source": "name", "match": "word", "position": 1, "pattern": "dress|dresses", "field": "type", "value": "Dress"},
  {"source": "name", "match": "word", "position": 1, "pattern": "skirt|skirts", "field": "type", "value": "Skirt"},
  {"source": "name", "match": "word", "position": 1, "pattern": "pyjama|pyjamas|pajama|pajamas", "field": "type", "value": "Pyjama"},
  {"source": "name", "match": "word", "position": 1, "pattern": "set|suit", "field": "type", "value": "Set"}
]'::jsonb
WHERE id = 1;
