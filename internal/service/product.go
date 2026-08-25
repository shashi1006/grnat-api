package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// ProductService manages the Funding OS preparedness solutions catalog and
// organizations' selections from it.
type ProductService struct {
	products repository.ProductRepo
}

// NewProductService creates a ProductService.
func NewProductService(products repository.ProductRepo) *ProductService {
	return &ProductService{products: products}
}

// ListProducts returns the active product catalog.
func (s *ProductService) ListProducts(ctx context.Context) ([]*domain.Product, error) {
	return s.products.List(ctx)
}

// GetProduct retrieves a single product by ID.
func (s *ProductService) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, err := s.products.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}
	return p, nil
}

// SaveSelection upserts an org's product selection.
func (s *ProductService) SaveSelection(ctx context.Context, params repository.SaveSelectionParams) (*domain.OrgProductSelection, error) {
	return s.products.SaveSelection(ctx, params)
}

// ListSelections lists an org's product selections.
func (s *ProductService) ListSelections(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgProductSelection, error) {
	return s.products.ListSelections(ctx, orgID)
}

// DeleteSelection removes an org's selection for a product.
func (s *ProductService) DeleteSelection(ctx context.Context, orgID, productID uuid.UUID) error {
	return s.products.DeleteSelection(ctx, orgID, productID)
}
