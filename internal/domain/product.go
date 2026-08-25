package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Product is a hardware/solution offering shown in the Funding OS wizard's
// "Recommended Preparedness Solutions" step. The nested configuration/add-on
// structure varies per product (fixed configuration cards vs. a base product
// with toggleable enhancements), so it is preserved verbatim in Catalog.
type Product struct {
	ID               uuid.UUID       `json:"id"`
	Slug             string          `json:"slug"`
	Name             string          `json:"name"`
	Category         *string         `json:"category,omitempty"`
	ShortDesc        *string         `json:"short_desc,omitempty"`
	Description      *string         `json:"description,omitempty"`
	SelectionType    *string         `json:"selection_type,omitempty"` // configurationCards | baseProductWithToggles
	PriceCents       int64           `json:"price_cents"`
	PriceType        string          `json:"price_type"`
	Featured         bool            `json:"featured"`
	FundingAlignment []string        `json:"funding_alignment"`
	Catalog          json.RawMessage `json:"catalog"` // configurations, add-ons, pricing examples, etc.
	IsActive         bool            `json:"is_active"`
	SortOrder        int32           `json:"sort_order"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// OrgProductSelection records an organization's chosen product configuration
// from the "Recommended Preparedness Solutions" step, including its computed
// price so downstream steps (funding roadmap, application package) can reuse
// the same project total without recomputing pricing logic server-side.
type OrgProductSelection struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductSlug     string    `json:"product_slug"`
	ConfigurationID *string   `json:"configuration_id,omitempty"`
	SelectedAddons  []string  `json:"selected_addons"`
	Quantity        int32     `json:"quantity"`
	UnitPriceCents  int64     `json:"unit_price_cents"`
	SubtotalCents   int64     `json:"subtotal_cents"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
