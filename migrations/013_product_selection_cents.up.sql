UPDATE org_product_selections
SET subtotal_cents = subtotal_cents * 100,
    unit_price_cents = unit_price_cents * 100;
