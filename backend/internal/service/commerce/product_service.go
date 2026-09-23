package commerce

import (
	"context"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type ProductService struct {
	productRepo domainCommerce.ProductRepository
}

func NewProductService(productRepo domainCommerce.ProductRepository) *ProductService {
	return &ProductService{productRepo: productRepo}
}

func (s *ProductService) ListActiveProducts(ctx context.Context) ([]*domainCommerce.Product, error) {
	return s.productRepo.ListActiveProductsWithPlans(ctx)
}

func (s *ProductService) GetProductByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Product, error) {
	return s.productRepo.GetProductByID(ctx, id)
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*domainCommerce.Product, error) {
	return s.productRepo.GetProductBySlug(ctx, slug)
}

func (s *ProductService) ListAllProducts(ctx context.Context) ([]*domainCommerce.Product, error) {
	return s.productRepo.ListAllProducts(ctx)
}

func (s *ProductService) CreateProduct(ctx context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return s.productRepo.CreateProduct(ctx, p)
}

func (s *ProductService) CreatePlan(ctx context.Context, plan *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	return s.productRepo.CreatePlan(ctx, plan)
}

func (s *ProductService) UpdateProduct(ctx context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	return s.productRepo.UpdateProduct(ctx, p)
}

func (s *ProductService) UpdatePlan(ctx context.Context, plan *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	return s.productRepo.UpdatePlan(ctx, plan)
}
