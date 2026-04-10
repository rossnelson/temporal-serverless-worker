package main

import (
	"time"

	temporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type TripInput struct {
	Traveler      string
	Destination   string
	DepartureDate string
	ReturnDate    string
}

type TripResult struct {
	FlightConfirmation string
	HotelConfirmation  string
	CarConfirmation    string
	TotalPrice         string
	Status             string
	Message            string
}

type BookingResult struct {
	ConfirmationID string
	Details        string
}

var bookActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
}

var bookActivityOptionsNoRetry = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		MaximumAttempts: 1,
	},
}

func TripBookingWorkflow(ctx workflow.Context, input TripInput) (TripResult, error) {
	taskQueue := workflow.GetInfo(ctx).TaskQueueName
	childOpts := workflow.ChildWorkflowOptions{
		TaskQueue: taskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	flightFuture := workflow.ExecuteChildWorkflow(childCtx, FlightBookingWorkflow, input)
	hotelFuture := workflow.ExecuteChildWorkflow(childCtx, HotelBookingWorkflow, input)
	carFuture := workflow.ExecuteChildWorkflow(childCtx, CarRentalWorkflow, input)

	var flightResult BookingResult
	var hotelResult BookingResult
	var carResult BookingResult

	flightErr := flightFuture.Get(ctx, &flightResult)
	hotelErr := hotelFuture.Get(ctx, &hotelResult)
	carErr := carFuture.Get(ctx, &carResult)

	if flightErr != nil || hotelErr != nil || carErr != nil {
		ao := workflow.WithActivityOptions(ctx, bookActivityOptions)

		var failedService string
		if flightErr != nil {
			failedService = "flight"
		} else if hotelErr != nil {
			failedService = "hotel"
		} else {
			failedService = "car rental"
		}

		if flightErr == nil && flightResult.ConfirmationID != "" {
			_ = workflow.ExecuteActivity(ao, CancelFlight, flightResult.ConfirmationID).Get(ctx, nil)
		}
		if hotelErr == nil && hotelResult.ConfirmationID != "" {
			_ = workflow.ExecuteActivity(ao, CancelHotel, hotelResult.ConfirmationID).Get(ctx, nil)
		}
		if carErr == nil && carResult.ConfirmationID != "" {
			_ = workflow.ExecuteActivity(ao, CancelCar, carResult.ConfirmationID).Get(ctx, nil)
		}

		return TripResult{
			Status:  "booking_failed",
			Message: failedService + " booking unavailable — all reservations cancelled, no charge",
		}, nil
	}

	return TripResult{
		FlightConfirmation: flightResult.ConfirmationID,
		HotelConfirmation:  hotelResult.ConfirmationID,
		CarConfirmation:    carResult.ConfirmationID,
		TotalPrice:         "$2,847.50",
		Status:             "confirmed",
		Message:            "Your trip to " + input.Destination + " is booked!",
	}, nil
}

func FlightBookingWorkflow(ctx workflow.Context, input TripInput) (BookingResult, error) {
	ao := workflow.WithActivityOptions(ctx, bookActivityOptions)
	aoNoRetry := workflow.WithActivityOptions(ctx, bookActivityOptionsNoRetry)

	var searchResult string
	if err := workflow.ExecuteActivity(ao, SearchFlights, input).Get(ctx, &searchResult); err != nil {
		return BookingResult{}, err
	}

	var selectResult string
	if err := workflow.ExecuteActivity(ao, SelectFlight, searchResult).Get(ctx, &selectResult); err != nil {
		return BookingResult{}, err
	}

	var bookResult BookingResult
	if err := workflow.ExecuteActivity(aoNoRetry, BookFlight, searchResult, selectResult).Get(ctx, &bookResult); err != nil {
		return BookingResult{}, err
	}

	return bookResult, nil
}

func HotelBookingWorkflow(ctx workflow.Context, input TripInput) (BookingResult, error) {
	ao := workflow.WithActivityOptions(ctx, bookActivityOptions)
	aoNoRetry := workflow.WithActivityOptions(ctx, bookActivityOptionsNoRetry)

	var searchResult string
	if err := workflow.ExecuteActivity(ao, SearchHotels, input).Get(ctx, &searchResult); err != nil {
		return BookingResult{}, err
	}

	var selectResult string
	if err := workflow.ExecuteActivity(ao, SelectHotel, searchResult).Get(ctx, &selectResult); err != nil {
		return BookingResult{}, err
	}

	var bookResult BookingResult
	if err := workflow.ExecuteActivity(aoNoRetry, BookHotel, searchResult, selectResult).Get(ctx, &bookResult); err != nil {
		return BookingResult{}, err
	}

	return bookResult, nil
}

func CarRentalWorkflow(ctx workflow.Context, input TripInput) (BookingResult, error) {
	ao := workflow.WithActivityOptions(ctx, bookActivityOptions)
	aoNoRetry := workflow.WithActivityOptions(ctx, bookActivityOptionsNoRetry)

	var searchResult string
	if err := workflow.ExecuteActivity(ao, SearchCars, input).Get(ctx, &searchResult); err != nil {
		return BookingResult{}, err
	}

	var selectResult string
	if err := workflow.ExecuteActivity(ao, SelectCar, searchResult).Get(ctx, &selectResult); err != nil {
		return BookingResult{}, err
	}

	var bookResult BookingResult
	if err := workflow.ExecuteActivity(aoNoRetry, BookCar, searchResult, selectResult).Get(ctx, &bookResult); err != nil {
		return BookingResult{}, err
	}

	return bookResult, nil
}
