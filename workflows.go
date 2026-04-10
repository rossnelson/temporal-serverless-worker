package main

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

type OrderInput struct {
	Customer   string
	Restaurant string
	Items      []string
}

type OrderResult struct {
	OrderID         string
	Total           float64
	TransactionID   string
	ReceiptURL      string
	DeliveryStatus  string
}

type PaymentResult struct {
	AuthCode      string
	TransactionID string
	ReceiptURL    string
}

type KitchenResult struct {
	Status string
}

var activityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
}

func FoodOrderWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {
	ao := workflow.WithActivityOptions(ctx, activityOptions)

	var menuResult MenuResult
	if err := workflow.ExecuteActivity(ao, BrowseMenu, input.Items).Get(ctx, &menuResult); err != nil {
		return OrderResult{}, err
	}

	var orderID string
	if err := workflow.ExecuteActivity(ao, PlaceOrder, input.Customer, menuResult.Items).Get(ctx, &orderID); err != nil {
		return OrderResult{}, err
	}

	taskQueue := workflow.GetInfo(ctx).TaskQueueName
	childOpts := workflow.ChildWorkflowOptions{
		TaskQueue: taskQueue,
	}

	paymentFuture := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(ctx, childOpts),
		PaymentWorkflow,
		orderID,
		menuResult.Total,
	)

	kitchenFuture := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(ctx, childOpts),
		KitchenWorkflow,
		orderID,
		input.Restaurant,
	)

	var paymentResult PaymentResult
	if err := paymentFuture.Get(ctx, &paymentResult); err != nil {
		return OrderResult{}, err
	}

	var kitchenResult KitchenResult
	if err := kitchenFuture.Get(ctx, &kitchenResult); err != nil {
		return OrderResult{}, err
	}

	var deliveryStatus string
	if err := workflow.ExecuteActivity(ao, DeliverOrder, orderID).Get(ctx, &deliveryStatus); err != nil {
		return OrderResult{}, err
	}

	return OrderResult{
		OrderID:        orderID,
		Total:          menuResult.Total,
		TransactionID:  paymentResult.TransactionID,
		ReceiptURL:     paymentResult.ReceiptURL,
		DeliveryStatus: deliveryStatus,
	}, nil
}

func PaymentWorkflow(ctx workflow.Context, orderID string, total float64) (PaymentResult, error) {
	ao := workflow.WithActivityOptions(ctx, activityOptions)

	var authCode string
	if err := workflow.ExecuteActivity(ao, AuthorizePayment, orderID, total).Get(ctx, &authCode); err != nil {
		return PaymentResult{}, err
	}

	var transactionID string
	if err := workflow.ExecuteActivity(ao, ChargePayment, orderID, authCode).Get(ctx, &transactionID); err != nil {
		return PaymentResult{}, err
	}

	var receiptURL string
	if err := workflow.ExecuteActivity(ao, SendReceipt, orderID, transactionID).Get(ctx, &receiptURL); err != nil {
		return PaymentResult{}, err
	}

	return PaymentResult{
		AuthCode:      authCode,
		TransactionID: transactionID,
		ReceiptURL:    receiptURL,
	}, nil
}

func KitchenWorkflow(ctx workflow.Context, orderID string, restaurant string) (KitchenResult, error) {
	ao := workflow.WithActivityOptions(ctx, activityOptions)

	if err := workflow.ExecuteActivity(ao, AcceptOrder, orderID, restaurant).Get(ctx, nil); err != nil {
		return KitchenResult{}, err
	}

	if err := workflow.ExecuteActivity(ao, PrepareFood, orderID).Get(ctx, nil); err != nil {
		return KitchenResult{}, err
	}

	if err := workflow.ExecuteActivity(ao, PackageOrder, orderID).Get(ctx, nil); err != nil {
		return KitchenResult{}, err
	}

	return KitchenResult{Status: "ready"}, nil
}
