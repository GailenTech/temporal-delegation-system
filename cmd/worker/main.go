package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-workflow/internal/activities"
	"temporal-workflow/internal/workflows"
)

const (
	taskQueue = "purchase-approval-task-queue"
)

func main() {
	log.Println("Starting Purchase Approval Worker...")

	// Create Temporal client
	// Read connection configuration from environment
	host := os.Getenv("TEMPORAL_HOST")
	port := os.Getenv("TEMPORAL_PORT")
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	
	// Set defaults if not provided
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "7233"
	}
	if namespace == "" {
		namespace = "default"
	}
	
	hostPort := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Connecting to Temporal at %s, namespace: %s", hostPort, namespace)
	
	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register Workflows
	w.RegisterWorkflow(workflows.PurchaseApprovalWorkflow)

	// Register Activities
	w.RegisterActivity(activities.ValidateAmazonProducts)
	w.RegisterActivity(activities.ExecuteAmazonPurchase)
	w.RegisterActivity(activities.GetRequiredApprovers)
	w.RegisterActivity(activities.NotifyEmployee)
	w.RegisterActivity(activities.NotifyResponsible)
	w.RegisterActivity(activities.CheckDuplicatePurchases)
	w.RegisterActivity(activities.LogPurchaseDecision)

	log.Println("Worker registered workflows and activities")
	log.Printf("Listening on task queue: %s", taskQueue)

	// Start worker in a goroutine
	go func() {
		err = w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatalln("Unable to start worker", err)
		}
	}()

	log.Println("Worker started successfully")
	log.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Worker shutting down...")
}