package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

func SearchFlights(ctx context.Context, input TripInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Searching flights", "destination", input.Destination)
	time.Sleep(3 * time.Second)
	return fmt.Sprintf("3 flights found to %s", input.Destination), nil
}

func SelectFlight(ctx context.Context, searchResult string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Selecting flight", "searchResult", searchResult)
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Selected: Delta DL-%d nonstop", time.Now().UnixNano()%1000), nil
}

func BookFlight(ctx context.Context, searchResult string, selectedFlight string) (BookingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Booking flight", "flight", selectedFlight)
	time.Sleep(4 * time.Second)
	if time.Now().UnixNano()%4 == 0 {
		return BookingResult{}, errors.New("flight booking failed: seat unavailable")
	}
	confirmationID := fmt.Sprintf("FL-%d", time.Now().UnixMilli()%100000)
	return BookingResult{ConfirmationID: confirmationID, Details: selectedFlight}, nil
}

func CancelFlight(ctx context.Context, confirmationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling flight", "confirmationID", confirmationID)
	time.Sleep(2 * time.Second)
	return nil
}

func SearchHotels(ctx context.Context, input TripInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Searching hotels", "destination", input.Destination)
	time.Sleep(3 * time.Second)
	return fmt.Sprintf("5 hotels found in %s", input.Destination), nil
}

func SelectHotel(ctx context.Context, searchResult string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Selecting hotel", "searchResult", searchResult)
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Selected: The Grand %s Hotel", searchResult), nil
}

func BookHotel(ctx context.Context, searchResult string, selectedHotel string) (BookingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Booking hotel", "hotel", selectedHotel)
	time.Sleep(4 * time.Second)
	if time.Now().UnixNano()%4 == 0 {
		return BookingResult{}, errors.New("hotel booking failed: no rooms available")
	}
	confirmationID := fmt.Sprintf("HT-%d", time.Now().UnixMilli()%100000)
	return BookingResult{ConfirmationID: confirmationID, Details: selectedHotel}, nil
}

func CancelHotel(ctx context.Context, confirmationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling hotel", "confirmationID", confirmationID)
	time.Sleep(2 * time.Second)
	return nil
}

func SearchCars(ctx context.Context, input TripInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Searching cars", "destination", input.Destination)
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("8 cars available in %s", input.Destination), nil
}

func SelectCar(ctx context.Context, searchResult string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Selecting car", "searchResult", searchResult)
	time.Sleep(1 * time.Second)
	return "Selected: Toyota Camry (mid-size)", nil
}

func BookCar(ctx context.Context, searchResult string, selectedCar string) (BookingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Booking car", "car", selectedCar)
	time.Sleep(3 * time.Second)
	if time.Now().UnixNano()%4 == 0 {
		return BookingResult{}, errors.New("car rental failed: vehicle unavailable")
	}
	confirmationID := fmt.Sprintf("CR-%d", time.Now().UnixMilli()%100000)
	return BookingResult{ConfirmationID: confirmationID, Details: selectedCar}, nil
}

func CancelCar(ctx context.Context, confirmationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling car rental", "confirmationID", confirmationID)
	time.Sleep(2 * time.Second)
	return nil
}
