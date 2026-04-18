CREATE TABLE IF NOT EXISTS attributes (
  name text primary key
);

CREATE TABLE IF NOT EXISTS attributes_values (
  id bigserial primary key,
  attribute text not null references attributes(name),
  value text not null,
  UNIQUE (attribute, value)
);

CREATE TABLE IF NOT EXISTS products_attributes (
  attribute text not null references attributes(name),
  attribute_value_id bigint not null references attributes_values(id),
  product_id bigint not null references products(id),
  PRIMARY KEY (attribute_value_id, product_id)
);

INSERT INTO attributes 
(name) VALUES 
  ('category'),      
  ('sub-category'),   
  ('brand'),         
  ('kind'),          
  ('type'),          
  ('unit'),          
  ('year'),          
  ('season'),        
  ('origin');        
