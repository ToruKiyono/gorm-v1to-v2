package main

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"example.com/legacy-shop/models"
	"example.com/legacy-shop/service"
)

func main() {
	db, err := gorm.Open("sqlite3", "legacy_shop.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.LogMode(true)
	db.AutoMigrate(&models.Product{}, &models.Tag{}, &models.StockLevel{}, &models.Promotion{}, &models.Customer{}, &models.Order{}, &models.OrderItem{})

	checkout := service.NewCheckoutService(db)
	if err := checkout.SeedData(); err != nil {
		panic(err)
	}

	order, err := checkout.Checkout("dev@example.com", []service.LineItem{{SKU: "coffee-beans", Quantity: 2}, {SKU: "ultra-laptop", Quantity: 1}})
	if err != nil {
		panic(err)
	}

	fmt.Printf("created order %d with %d items, total %d cents\n", order.ID, len(order.Items), order.TotalCents)

	orders, err := checkout.OrdersForCustomer("dev@example.com")
	if err != nil {
		panic(err)
	}

	fmt.Printf("customer has %d historical orders\n", len(orders))
}
