package main

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ConfirmationInput struct {
	Customer   string
	Restaurant string
	OrderID    string
}

type ConfirmationResult struct {
	OrderID string
	Status  string
	Message string
}

func OrderConfirmationWorkflow(ctx workflow.Context, input ConfirmationInput) (ConfirmationResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var submitResult string
	if err := workflow.ExecuteActivity(ctx, SubmitOrderToRestaurant, input).Get(ctx, &submitResult); err != nil {
		return ConfirmationResult{}, err
	}

	if err := workflow.Sleep(ctx, 15*time.Second); err != nil {
		return ConfirmationResult{}, err
	}

	var confirmed bool
	if err := workflow.ExecuteActivity(ctx, CheckRestaurantConfirmation, input.OrderID).Get(ctx, &confirmed); err != nil {
		return ConfirmationResult{}, err
	}

	if confirmed {
		if err := workflow.ExecuteActivity(ctx, NotifyCustomer, input.Customer, "confirmed", input.OrderID).Get(ctx, nil); err != nil {
			return ConfirmationResult{}, err
		}
		return ConfirmationResult{
			OrderID: input.OrderID,
			Status:  "confirmed",
			Message: fmt.Sprintf("%s confirmed your order!", input.Restaurant),
		}, nil
	}

	if err := workflow.ExecuteActivity(ctx, EscalateToManager, input.Restaurant, input.OrderID).Get(ctx, nil); err != nil {
		return ConfirmationResult{}, err
	}

	if err := workflow.Sleep(ctx, 10*time.Second); err != nil {
		return ConfirmationResult{}, err
	}

	if err := workflow.ExecuteActivity(ctx, CheckRestaurantConfirmation, input.OrderID).Get(ctx, &confirmed); err != nil {
		return ConfirmationResult{}, err
	}

	if confirmed {
		if err := workflow.ExecuteActivity(ctx, NotifyCustomer, input.Customer, "confirmed_after_escalation", input.OrderID).Get(ctx, nil); err != nil {
			return ConfirmationResult{}, err
		}
		return ConfirmationResult{
			OrderID: input.OrderID,
			Status:  "confirmed_after_escalation",
			Message: "Order confirmed after escalation — sorry for the wait!",
		}, nil
	}

	if err := workflow.ExecuteActivity(ctx, CancelOrder, input.OrderID).Get(ctx, nil); err != nil {
		return ConfirmationResult{}, err
	}

	var refundID string
	if err := workflow.ExecuteActivity(ctx, IssueRefund, input.Customer, input.OrderID).Get(ctx, &refundID); err != nil {
		return ConfirmationResult{}, err
	}

	if err := workflow.ExecuteActivity(ctx, NotifyCustomer, input.Customer, "cancelled", input.OrderID).Get(ctx, nil); err != nil {
		return ConfirmationResult{}, err
	}

	return ConfirmationResult{
		OrderID: input.OrderID,
		Status:  "cancelled",
		Message: fmt.Sprintf("%s didn't respond — order cancelled and refund issued", input.Restaurant),
	}, nil
}
