UPDATE products SET catalog = catalog - 'baseProjectCost' - 'traumaConfiguration' - 'optionalEnhancements'
WHERE slug IN ('acs-digital-wall-station', 'acs-mobile-digital-station', 'acs-mobile-event-pod');
