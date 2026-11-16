// The products list's query string from its filter panel. Only filters that
// are set are sent; ServerSideTable skips empty values, but keeping the
// object lean avoids needless URL churn. source 'label_photo' lists the
// products the label reader made ("From photo").
export function productQueryParams (filters) {
  const f = filters || {}
  const out = {}
  if (f.is_active === true || f.is_active === false) out.is_active = String(f.is_active)
  if (f.has_variants === true) out.has_variants = 'true'
  if (f.source) out.source = f.source
  if (f.created_from) out.created_from = f.created_from
  if (f.created_to) out.created_to = f.created_to
  if (Array.isArray(f.attribute_value_ids) && f.attribute_value_ids.length > 0) {
    out.attribute_value_ids = f.attribute_value_ids
  }
  return out
}

// A product the label reader made gets a "From photo" tag in the list.
export function isFromPhoto (product) {
  return product?.source === 'label_photo'
}
