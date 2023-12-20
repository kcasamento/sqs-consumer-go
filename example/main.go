package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	consumer "github.com/kcasamento/sqs-consumer-go"
)

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	queueUrl := "http://localhost:4566/000000000000/consumer-test-queue"
	c, err := consumer.New(
		queueUrl,
		func(_ context.Context, processId string, _ interface{}) (bool, error) {
			time.Sleep(2 * time.Second)
			fmt.Printf("processId: %s\n", processId)
			return false, nil
		},
		consumer.WithHeartbeat(1*time.Second),
		consumer.WithConcurrency(10),
		consumer.WithMaxIdleTime(0),
		consumer.WithEndpointResolver(&SqsLocalConfig{
			Endpoint: "http://localhost:4566",
		}),
		consumer.WithCredentialProvider(&LocalCredConfig{}),
	)
	if err != nil {
		log.Printf("error creating consumer: %v", err)
	}

	ctx := context.Background()
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.Start(cancelCtx)
	log.Printf("consuming is running...")

	<-done

	c.Stop(ctx)

	log.Printf("consumer is shutdown")
}
