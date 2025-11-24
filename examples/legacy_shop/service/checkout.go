package service

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	"example.com/legacy-shop/models"
)

// LineItem is a user request describing a purchase line.
type LineItem struct {
	SKU      string
	Quantity int
}

// CheckoutService bundles shopping workflows using GORM v1 APIs.
type CheckoutService struct {
	db *gorm.DB
}

// NewCheckoutService wires the DB dependency.
func NewCheckoutService(db *gorm.DB) *CheckoutService {
	return &CheckoutService{db: db}
}

// SeedData populates products, stock, and tags in a transaction.
func (s *CheckoutService) SeedData() error {
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	laptop := models.Product{
		SKU:        "ultra-laptop",
		Name:       "Ultra Laptop",
		PriceCents: 199999,
		Tags: []models.Tag{
			{Name: "electronics"},
			{Name: "premium"},
		},
		Stock:      models.StockLevel{Quantity: 5, UpdatedBy: "system"},
		Promotions: []models.Promotion{{Title: "Spring Sale", Discount: 5.0, Active: true}},
	}

	coffee := models.Product{
		SKU:        "coffee-beans",
		Name:       "Single Origin Beans",
		PriceCents: 3299,
		Tags:       []models.Tag{{Name: "grocery"}},
		Stock:      models.StockLevel{Quantity: 40, UpdatedBy: "system"},
	}

	products := []*models.Product{&laptop, &coffee}
	for _, p := range products {
		if err := tx.Where(models.Product{SKU: p.SKU}).Assign(p).FirstOrCreate(p).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// Checkout converts requested lines into a persisted order with stock updates.
func (s *CheckoutService) Checkout(email string, lines []LineItem) (*models.Order, error) {
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}

	var customer models.Customer
	if err := tx.Where("lower(email) = ?", strings.ToLower(email)).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			customer = models.Customer{Email: strings.ToLower(email), Name: email}
			if err := tx.Create(&customer).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		} else {
			tx.Rollback()
			return nil, err
		}
	}

	order := models.Order{CustomerID: customer.ID, Status: "draft"}

	for _, line := range lines {
		var product models.Product
		if err := tx.Preload("Promotions").Preload("Tags").Preload("Stock").Where("sku = ?", line.SKU).First(&product).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if product.Stock.Quantity < line.Quantity {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient stock for %s", product.SKU)
		}

		lineTotal := line.Quantity * product.PriceCents
		order.TotalCents += lineTotal
		order.Items = append(order.Items, models.OrderItem{ProductID: product.ID, Quantity: line.Quantity, Subtotal: lineTotal})

		if err := tx.Model(&product.Stock).Updates(map[string]interface{}{
			"quantity":   gorm.Expr("quantity - ?", line.Quantity),
			"updated_by": "checkout",
		}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	order.Status = "submitted"
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload with joins to demonstrate Preload behavior pre-migration.
	if err := s.db.Preload("Items").Preload("Items.Product").Preload("Items.Product.Tags").Preload("Customer").First(&order, order.ID).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

// OrdersForCustomer shows a query-only flow using v1 chainable scopes.
func (s *CheckoutService) OrdersForCustomer(email string) ([]models.Order, error) {
	var orders []models.Order
	err := s.db.Where("email = ?", strings.ToLower(email)).
		Model(&models.Customer{}).
		Related(&orders).
		Preload("Items").
		Preload("Items.Product").
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}
