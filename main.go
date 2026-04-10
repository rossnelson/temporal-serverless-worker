package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	defaultHostPort       = "127.0.0.1:7233"
	defaultNamespace      = "default"
	defaultTaskQueueName  = "worker-versioning-sample"
	defaultDeploymentName = "test"
	defaultBuildID        = "v1"
)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, event json.RawMessage) error {
	logger := MakeLogger()
	defer func() { _ = logger.Sync() }()

	hostPort := getEnvDefault("HOST_PORT", defaultHostPort)
	namespace := getEnvDefault("NAMESPACE", defaultNamespace)
	taskQueueName := getEnvDefault("TQ_NAME", defaultTaskQueueName)
	deploymentName := getEnvDefault("DEPLOYMENT_NAME", defaultDeploymentName)
	buildID := getEnvDefault("BUILD_ID", defaultBuildID)
	enableTLS := os.Getenv("ENABLE_TLS")
	tlsKey := os.Getenv("TLS_KEY")
	tlsCert := os.Getenv("TLS_CERT")
	apiKey := os.Getenv("API_KEY")

	var tlsConfig *tls.Config
	var credentials client.Credentials

	if enableTLS != "" {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}

		config, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}
		svc := secretsmanager.NewFromConfig(config)

		if tlsKey != "" && tlsCert != "" {
			clientCert, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &tlsCert})
			if err != nil {
				return err
			}
			clientKey, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &tlsKey})
			if err != nil {
				return err
			}

			cert, err := tls.X509KeyPair([]byte(*clientCert.SecretString), []byte(*clientKey.SecretString))
			if err != nil {
				return err
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
			logger.Info("Configured client cert", zap.String("cert", fmt.Sprintf("%#v", cert)))
		}

		if apiKey != "" {
			apiKeyValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &apiKey})
			if err != nil {
				return err
			}
			credentials = client.NewAPIKeyStaticCredentials(*apiKeyValue.SecretString)
		}
	}

	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
		Logger:    newZapSDKLogger(logger),
		ConnectionOptions: client.ConnectionOptions{
			TLS: tlsConfig,
		},
		Credentials:             credentials,
		WorkerHeartbeatInterval: 10 * time.Second,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	wOpts := worker.Options{
		DisableEagerActivities: true,
		DeploymentOptions: worker.DeploymentOptions{
			UseVersioning: true,
			Version: worker.WorkerDeploymentVersion{
				DeploymentName: deploymentName,
				BuildID:        buildID,
			},
			DefaultVersioningBehavior: workflow.VersioningBehaviorPinned,
		},

		MaxConcurrentActivityExecutionSize:     2,
		MaxConcurrentWorkflowTaskExecutionSize: 2,
		MaxConcurrentNexusTaskExecutionSize:    2,
	}

	w := worker.New(c, taskQueueName, wOpts)
	w.RegisterWorkflow(FoodOrderWorkflow)
	w.RegisterWorkflow(PaymentWorkflow)
	w.RegisterWorkflow(KitchenWorkflow)
	w.RegisterActivity(BrowseMenu)
	w.RegisterActivity(PlaceOrder)
	w.RegisterActivity(AuthorizePayment)
	w.RegisterActivity(ChargePayment)
	w.RegisterActivity(SendReceipt)
	w.RegisterActivity(AcceptOrder)
	w.RegisterActivity(PrepareFood)
	w.RegisterActivity(PackageOrder)
	w.RegisterActivity(DeliverOrder)
	w.RegisterWorkflow(TripBookingWorkflow)
	w.RegisterWorkflow(FlightBookingWorkflow)
	w.RegisterWorkflow(HotelBookingWorkflow)
	w.RegisterWorkflow(CarRentalWorkflow)
	w.RegisterActivity(SearchFlights)
	w.RegisterActivity(SelectFlight)
	w.RegisterActivity(BookFlight)
	w.RegisterActivity(CancelFlight)
	w.RegisterActivity(SearchHotels)
	w.RegisterActivity(SelectHotel)
	w.RegisterActivity(BookHotel)
	w.RegisterActivity(CancelHotel)
	w.RegisterActivity(SearchCars)
	w.RegisterActivity(SelectCar)
	w.RegisterActivity(BookCar)
	w.RegisterActivity(CancelCar)
	w.RegisterWorkflow(OrderConfirmationWorkflow)
	w.RegisterActivity(SubmitOrderToRestaurant)
	w.RegisterActivity(CheckRestaurantConfirmation)
	w.RegisterActivity(EscalateToManager)
	w.RegisterActivity(CancelOrder)
	w.RegisterActivity(IssueRefund)
	w.RegisterActivity(NotifyCustomer)

	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt, syscall.SIGTERM)
	ret := make(chan any, 1)
	go func() {
		s := <-osSignalChannel
		ret <- s
		close(ret)
	}()
	time.AfterFunc(1*time.Minute, func() { ret <- "timed out" })

	logger.Info("subprocess worker starting",
		zap.String("hostPort", hostPort),
		zap.String("namespace", namespace),
		zap.String("taskQueue", taskQueueName),
	)
	if err := w.Run(ret); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func getEnvDefault(env, defaultVal string) string {
	envValue := os.Getenv(env)
	if envValue == "" {
		return defaultVal
	}
	return envValue
}
