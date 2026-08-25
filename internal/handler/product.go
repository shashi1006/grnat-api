package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// ProductHandler handles the preparedness solutions catalog and org selections.
type ProductHandler struct {
	svc *service.ProductService
}

// NewProductHandler creates a ProductHandler.
func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// ListProducts godoc
// @Summary      List the Funding OS preparedness solutions catalog
// @Tags         products
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Router       /products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	products, err := h.svc.ListProducts(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, products)
}

// GetProduct godoc
// @Summary      Get a single product by ID
// @Tags         products
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Product UUID"
// @Success      200  {object}  response.Envelope
// @Router       /products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	product, err := h.svc.GetProduct(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "product not found")
		return
	}
	response.OK(c, product)
}

// SaveProductSelection godoc
// @Summary      Save an org's product/configuration selection
// @Tags         products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string                      true  "Organization UUID"
// @Param        body  body  saveSelectionRequest         true  "Selection data"
// @Success      200   {object}  response.Envelope
// @Router       /orgs/{id}/product-selection [put]
func (h *ProductHandler) SaveProductSelection(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org id")
		return
	}
	var req saveSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		response.BadRequest(c, "invalid product_id")
		return
	}
	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	selection, err := h.svc.SaveSelection(c.Request.Context(), repository.SaveSelectionParams{
		OrgID:           orgID,
		ProductID:       productID,
		ConfigurationID: req.ConfigurationID,
		SelectedAddons:  req.SelectedAddons,
		Quantity:        quantity,
		UnitPriceCents:  req.UnitPriceCents,
		SubtotalCents:   req.SubtotalCents,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, selection)
}

// ListProductSelections godoc
// @Summary      List an org's product selections
// @Tags         products
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Organization UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{id}/product-selection [get]
func (h *ProductHandler) ListProductSelections(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org id")
		return
	}
	selections, err := h.svc.ListSelections(c.Request.Context(), orgID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, selections)
}

type saveSelectionRequest struct {
	ProductID       string   `json:"product_id" binding:"required"`
	ConfigurationID *string  `json:"configuration_id"`
	SelectedAddons  []string `json:"selected_addons"`
	Quantity        int32    `json:"quantity"`
	UnitPriceCents  int64    `json:"unit_price_cents"`
	SubtotalCents   int64    `json:"subtotal_cents"`
}
