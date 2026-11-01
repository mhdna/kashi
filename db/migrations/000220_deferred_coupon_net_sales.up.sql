-- Net sales leave out "calculate on checkout" coupons, as kashi-pos does: such
-- a coupon is settled later by whoever issued it, so the part of an invoice it
-- paid is not income. The coupon a till redeemed is found through
-- coupons.redeemed_invoice_ref (the invoice's client_ref); the amount is the
-- invoice's "On Coupon" payment line. Counted coupons (any other type, or none
-- known) change nothing. Never more than the invoice's total (a coupon worth
-- more than the sale).
CREATE INDEX IF NOT EXISTS idx_coupons_redeemed_invoice_ref
  ON coupons (redeemed_invoice_ref) WHERE redeemed_invoice_ref IS NOT NULL;

CREATE OR REPLACE FUNCTION branch_invoice_deferred_coupons(p_invoice_id bigint)
RETURNS bigint
LANGUAGE sql STABLE
AS $$
  SELECT LEAST(COALESCE(sum(p.amount), 0), GREATEST((SELECT grand_total FROM branch_invoices WHERE id = p_invoice_id), 0))::bigint
  FROM branch_invoice_payments p
  WHERE p.branch_invoice_id = p_invoice_id
    AND lower(p.account_name) = 'on coupon'
    AND EXISTS (
      SELECT 1
      FROM branch_invoices bi
      JOIN coupons c ON c.redeemed_invoice_ref = bi.client_ref
      JOIN coupon_types ct ON ct.id = c.type_id
      WHERE bi.id = p_invoice_id AND ct.calculate_on_checkout
    )
$$;
