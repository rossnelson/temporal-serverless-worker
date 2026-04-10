package main

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

type MenuItem struct {
	Name  string
	Price float64
}

type MenuResult struct {
	Items []MenuItem
	Total float64
}

func BrowseMenu(ctx context.Context, items []string) (MenuResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Browsing menu and selecting items")
	time.Sleep(3 * time.Second)

	menu := []MenuItem{
		{Name: "Spicy Tuna Roll", Price: 14.99},
		{Name: "Margherita Pizza", Price: 18.50},
		{Name: "Street Tacos (3)", Price: 12.00},
		{Name: "Smash Burger", Price: 15.75},
		{Name: "Miso Soup", Price: 4.50},
	}

	var total float64
	for _, item := range menu {
		total += item.Price
	}

	return MenuResult{Items: menu, Total: total}, nil
}

func PlaceOrder(ctx context.Context, customer string, items []MenuItem) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Placing order", "customer", customer)
	time.Sleep(3 * time.Second)

	orderID := fmt.Sprintf("ORDER-%d", time.Now().Unix())
	return orderID, nil
}

func AuthorizePayment(ctx context.Context, orderID string, total float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Authorizing payment", "orderID", orderID, "total", total)
	time.Sleep(1 * time.Second)

	return fmt.Sprintf("AUTH-%d", time.Now().UnixMilli()%10000), nil
}

func ChargePayment(ctx context.Context, orderID string, authCode string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Charging payment", "orderID", orderID, "authCode", authCode)
	time.Sleep(1 * time.Second)

	return fmt.Sprintf("TXN-%d", time.Now().UnixMilli()%100000), nil
}

func SendReceipt(ctx context.Context, orderID string, transactionID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending receipt", "orderID", orderID, "transactionID", transactionID)
	time.Sleep(1 * time.Second)

	return fmt.Sprintf("https://receipts.example.com/orders/%s/receipt/%s", orderID, transactionID), nil
}

func AcceptOrder(ctx context.Context, orderID string, restaurant string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Restaurant accepting order", "orderID", orderID, "restaurant", restaurant)
	time.Sleep(1 * time.Second)

	return nil
}

func PrepareFood(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Kitchen preparing food", "orderID", orderID)
	time.Sleep(20 * time.Second)

	return nil
}

func PackageOrder(ctx context.Context, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Packaging order for delivery", "orderID", orderID)
	time.Sleep(1 * time.Second)

	return nil
}

func DeliverOrder(ctx context.Context, orderID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Delivering order", "orderID", orderID)
	time.Sleep(15 * time.Second)

	return "delivered", nil
}
