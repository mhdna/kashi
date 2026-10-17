-- Arabic branch-scoped expense categories, so every branch's expense picker
-- (synced down to kashi-pos) has the usual shop costs out of the box.
-- Icons are limited to ones kashi-pos registers in its lib/icons.ts --
-- an unregistered mdi name renders as raw text on the till.
--
-- ON CONFLICT DO NOTHING: a name an admin already created (كهرباء, for
-- one) keeps whatever scope / icon / active state it was given.
insert into expense_categories (name, icon, color, is_active, scope) values
  ('كهرباء',              'mdi-flash',                  'amber',           true, 'branch'),
  ('مياه',                'mdi-water',                  'blue',            true, 'branch'),
  ('إيجار',               'mdi-home-city',              'deep-purple',     true, 'branch'),
  ('أجور يومية',          'mdi-cash-multiple',          'green',           true, 'branch'),
  ('إنترنت',              'mdi-wifi',                   'cyan',            true, 'branch'),
  ('هاتف',                'mdi-phone',                  'teal',            true, 'branch'),
  ('صيانة',               'mdi-wrench',                 'blue-grey',       true, 'branch'),
  ('تنظيف',               'mdi-broom',                  'light-green',     true, 'branch'),
  ('قرطاسية ومستلزمات',   'mdi-package-variant-closed', 'orange-darken-2', true, 'branch'),
  ('أكياس وتغليف',        'mdi-gift-outline',           'pink',            true, 'branch'),
  ('ورق طباعة وإيصالات',  'mdi-printer',                'grey',            true, 'branch'),
  ('نقل وتوصيل',          'mdi-truck',                  'indigo',          true, 'branch'),
  ('محروقات',             'mdi-gas-station',            'deep-orange',     true, 'branch'),
  ('ضيافة',               'mdi-coffee',                 'brown',           true, 'branch'),
  ('طعام للموظفين',       'mdi-food',                   'orange',          true, 'branch'),
  ('تسويق وإعلان',        'mdi-bullhorn',               'purple',          true, 'branch'),
  ('مصاريف متفرقة',       'mdi-tag-outline',            'blue-grey',       true, 'branch')
on conflict (name) do nothing;
