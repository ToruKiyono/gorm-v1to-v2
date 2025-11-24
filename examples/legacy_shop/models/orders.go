package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// Customer represents an end-user placing orders.
type Customer struct {
	gorm.Model
	Email  string `gorm:"type:varchar(255);unique_index"`
	Name   string `gorm:"type:varchar(255)"`
	Orders []Order
}

// OrderItem captures each product line in an order.
type OrderItem struct {
	gorm.Model
	OrderID   uint
	ProductID uint
	Quantity  int
	Subtotal  int
	Product   Product
}

// Order aggregates purchased items.
type Order struct {
	gorm.Model
	CustomerID uint
	Status     string `gorm:"type:varchar(64)"`
	Items      []OrderItem
	TotalCents int
	Notes      *string `gorm:"type:text"`
}

// BeforeSave ensures totals stay non-negative.
func (o *Order) BeforeSave(scope *gorm.Scope) error {
	if o.TotalCents < 0 {
		return fmt.Errorf("total must be non-negative")
	}
	return nil
}
