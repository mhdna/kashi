CREATE TABLE IF NOT EXISTS attributes (
  id bigserial primary key,
  name text not null unique
);

CREATE TABLE IF NOT EXISTS attributes_values (
  id bigserial primary key,
  attribute_id bigint not null references attributes(id) on delete cascade,
  value text not null unique
);

CREATE TABLE IF NOT EXISTS products_attributes (
  attribute_id bigint not null references attributes(id),
  attribute_value_id bigint not null references attributes_values(id),
  product_id bigint not null references products(id),
  PRIMARY KEY (attribute_value_id, product_id)
);

INSERT INTO attributes 
(id, name) VALUES 
  (1, 'category'),      
  (2, 'subcategory'),   
  (3, 'brand'),         
  (4, 'kind'),          
  (5, 'type'),          
  (6, 'unit'),          
  (7, 'year'),          
  (8, 'season'),        
  (9, 'origin');        
