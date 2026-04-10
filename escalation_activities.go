package main

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

func SubmitOrderToRestaurant(ctx context.Context, input ConfirmationInput) (string, error) {
	activity.GetLogger(ctx).Info("Submitting order to restaurant", "orderID", input.OrderID, "restaurant", input.Restaurant)
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Order %s submitted to %s", input.OrderID, input.Restaurant), nil
}

func CheckRestaurantConfirmation(ctx context.Context, orderID string) (bool, error) {
	activity.GetLogger(ctx).Info("Checking restaurant confirmation", "orderID", orderID)
	time.Sleep(2 * time.Second)
	confirmed := time.Now().UnixNano()%2 == 0
	return confirmed, nil
}

func EscalateToManager(ctx context.Context, restaurant, orderID string) error {
	activity.GetLogger(ctx).Info(fmt.Sprintf("Escalating %s at %s to manager", orderID, restaurant))
	time.Sleep(2 * time.Second)
	return nil
}

func CancelOrder(ctx context.Context, orderID string) error {
	activity.GetLogger(ctx).Info("Cancelling order", "orderID", orderID)
	time.Sleep(2 * time.Second)
	return nil
}

func IssueRefund(ctx context.Context, customer, orderID string) (string, error) {
	activity.GetLogger(ctx).Info("Issuing refund", "customer", customer, "orderID", orderID)
	time.Sleep(2 * time.Second)
	refundID := fmt.Sprintf("REFUND-%d", time.Now().UnixNano()%10000)
	return refundID, nil
}

func NotifyCustomer(ctx context.Context, customer, status, orderID string) error {
	activity.GetLogger(ctx).Info("Notifying customer", "customer", customer, "status", status, "orderID", orderID)
	time.Sleep(1 * time.Second)
	return nil
}
