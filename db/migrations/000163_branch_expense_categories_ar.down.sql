-- Removes the seeded categories that nothing has used yet; any already
-- picked on an expense stay, since those rows reference them.
delete from expense_categories ec
where ec.name in (
  'مياه', 'إيجار', 'أجور يومية', 'إنترنت', 'هاتف', 'صيانة',
  'تنظيف', 'قرطاسية ومستلزمات', 'أكياس وتغليف', 'ورق طباعة وإيصالات',
  'نقل وتوصيل', 'محروقات', 'ضيافة', 'طعام للموظفين', 'تسويق وإعلان',
  'مصاريف متفرقة'
)
  and not exists (select 1 from expenses e where e.category_id = ec.id)
  and not exists (select 1 from branch_expenses be where be.category_id = ec.id)
  and not exists (select 1 from recurring_expenses re where re.category_id = ec.id);
