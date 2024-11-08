package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	consumer "github.com/kcasamento/sqs-consumer-go/consumer"
	"github.com/kcasamento/sqs-consumer-go/internal/runner"

	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	queueUrl := "http://localhost:4566/000000000000/consumer-test-queue"
	c, sh, err := consumer.New(
		queueUrl,
		func(_ context.Context, processId string, _ awstypes.Message) (bool, error) {
			// time.Sleep(2 * time.Second)
			// fmt.Printf("processId: %s\n", processId)
			return false, nil
		},
		consumer.WithHeartbeat(1*time.Second),
		consumer.WithConcurrency(10),
		consumer.WithMaxIdleTime(5),
		consumer.WithEndpointResolver(&SqsLocalConfig{
			Endpoint: "http://localhost:4566",
		}),
		consumer.WithCredentialProvider(&LocalCredConfig{}),
		consumer.WithMaxMessages(10),
		consumer.WithDispatchStrategy(runner.WorkerPool),
		// consumer.WithDispatchStrategy(runner.SemPool),
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

	_ = sh(ctx)

	log.Printf("consumer is shutdown")
}
