package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// Product models a sellable SKU with relational edges for stock and tags.
type Product struct {
	gorm.Model
	SKU        string      `gorm:"type:varchar(64);unique_index"`
	Name       string      `gorm:"type:varchar(255)"`
	PriceCents int         `gorm:"type:int"`
	Tags       []Tag       `gorm:"many2many:product_tags"`
	Stock      StockLevel  `gorm:"foreignkey:ProductID"`
	Promotions []Promotion `gorm:"foreignkey:ProductID"`
}

// BeforeSave validates product pricing before persistence.
func (p *Product) BeforeSave(scope *gorm.Scope) error {
	if p.PriceCents < 0 {
		return fmt.Errorf("price must be positive")
	}
	return nil
}

// Tag represents a simple label for grouping products.
type Tag struct {
	gorm.Model
	Name string `gorm:"type:varchar(128);unique_index"`
}

// StockLevel captures on-hand quantities for a product.
type StockLevel struct {
	gorm.Model
	ProductID uint
	Quantity  int    `gorm:"type:int"`
	UpdatedBy string `gorm:"type:varchar(64)"`
}

// Promotion records simple percentage discounts for a SKU.
type Promotion struct {
	gorm.Model
	ProductID uint
	Title     string  `gorm:"type:varchar(200)"`
	Discount  float32 `gorm:"type:decimal(5,2)"`
	Active    bool
}
