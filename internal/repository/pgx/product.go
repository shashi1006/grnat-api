package pgxrepo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type productRepo struct {
	db *pgxpool.Pool
}

// NewProductRepo creates a pgx-backed ProductRepo.
func NewProductRepo(db *pgxpool.Pool) repository.ProductRepo {
	return &productRepo{db: db}
}

const productCols = `id, slug, name, category, short_desc, description, selection_type,
	price_cents, price_type, featured, funding_alignment, catalog, is_active, sort_order,
	created_at, updated_at`

func (r *productRepo) Create(ctx context.Context, p repository.CreateProductParams) (*domain.Product, error) {
	const q = `
		INSERT INTO products (slug, name, category, short_desc, description, selection_type,
			price_cents, price_type, featured, funding_alignment, catalog, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING ` + productCols
	row := r.db.QueryRow(ctx, q,
		p.Slug, p.Name, p.Category, p.ShortDesc, p.Description, p.SelectionType,
		p.PriceCents, p.PriceType, p.Featured, p.FundingAlignment, p.Catalog, p.SortOrder,
	)
	return scanProduct(row)
}

func (r *productRepo) List(ctx context.Context) ([]*domain.Product, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+productCols+` FROM products WHERE is_active = TRUE ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var products []*domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *productRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return scanProduct(r.db.QueryRow(ctx, `SELECT `+productCols+` FROM products WHERE id=$1`, id))
}

func (r *productRepo) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	return scanProduct(r.db.QueryRow(ctx, `SELECT `+productCols+` FROM products WHERE slug=$1`, slug))
}

func (r *productRepo) SaveSelection(ctx context.Context, p repository.SaveSelectionParams) (*domain.OrgProductSelection, error) {
	const q = `
		INSERT INTO org_product_selections (org_id, product_id, configuration_id, selected_addons, quantity, unit_price_cents, subtotal_cents)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (org_id, product_id) DO UPDATE SET
			configuration_id = EXCLUDED.configuration_id,
			selected_addons  = EXCLUDED.selected_addons,
			quantity         = EXCLUDED.quantity,
			unit_price_cents = EXCLUDED.unit_price_cents,
			subtotal_cents   = EXCLUDED.subtotal_cents,
			updated_at       = NOW()
		RETURNING id, org_id, product_id, configuration_id, selected_addons, quantity,
		          unit_price_cents, subtotal_cents, created_at, updated_at`
	row := r.db.QueryRow(ctx, q,
		p.OrgID, p.ProductID, p.ConfigurationID, p.SelectedAddons, p.Quantity, p.UnitPriceCents, p.SubtotalCents,
	)
	return scanSelection(row)
}

func (r *productRepo) ListSelections(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgProductSelection, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.id, s.org_id, s.product_id, s.configuration_id, s.selected_addons, s.quantity,
		        s.unit_price_cents, s.subtotal_cents, s.created_at, s.updated_at, p.slug
		 FROM org_product_selections s
		 JOIN products p ON p.id = s.product_id
		 WHERE s.org_id = $1 ORDER BY s.created_at ASC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list product selections: %w", err)
	}
	defer rows.Close()
	var out []*domain.OrgProductSelection
	for rows.Next() {
		var s domain.OrgProductSelection
		if err := rows.Scan(&s.ID, &s.OrgID, &s.ProductID, &s.ConfigurationID, &s.SelectedAddons,
			&s.Quantity, &s.UnitPriceCents, &s.SubtotalCents, &s.CreatedAt, &s.UpdatedAt, &s.ProductSlug); err != nil {
			return nil, fmt.Errorf("scan product selection: %w", err)
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *productRepo) DeleteSelection(ctx context.Context, orgID, productID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM org_product_selections WHERE org_id=$1 AND product_id=$2`, orgID, productID)
	return err
}

func scanProduct(row scannable) (*domain.Product, error) {
	var p domain.Product
	var priceType string
	var catalog []byte
	err := row.Scan(
		&p.ID, &p.Slug, &p.Name, &p.Category, &p.ShortDesc, &p.Description, &p.SelectionType,
		&p.PriceCents, &priceType, &p.Featured, &p.FundingAlignment, &catalog, &p.IsActive, &p.SortOrder,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan product: %w", err)
	}
	p.PriceType = priceType
	p.Catalog = catalog
	return &p, nil
}

func scanSelection(row scannable) (*domain.OrgProductSelection, error) {
	var s domain.OrgProductSelection
	err := row.Scan(&s.ID, &s.OrgID, &s.ProductID, &s.ConfigurationID, &s.SelectedAddons,
		&s.Quantity, &s.UnitPriceCents, &s.SubtotalCents, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan product selection: %w", err)
	}
	return &s, nil
}
