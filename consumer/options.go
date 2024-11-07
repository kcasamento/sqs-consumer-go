package consumer

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/kcasamento/sqs-consumer-go/internal/runner"
	"github.com/kcasamento/sqs-consumer-go/internal/service"
)

type ConsumerOptions func(*Consumer)

// Provide your own sqs client to interface with sqs
func WithSqsClient(client *sqs.Client) ConsumerOptions {
	return func(c *Consumer) {
		c.client = service.NewSqsClient(client)
	}
}

func WithConcurrency(concurrency int) ConsumerOptions {
	return func(c *Consumer) {
		c.concurrency = concurrency
	}
}

// Configures the consumer to send a heartbeat to sqs
// for long running messages and update their visibility timeout
func WithHeartbeat(interval time.Duration) ConsumerOptions {
	return func(c *Consumer) {
		c.heartbeatConfig = &heartbeatConfig{
			interval: interval,
		}
	}
}

// Sets the max messages to pull in at a time
// Default: 10
func WithMaxMessages(max int) ConsumerOptions {
	return func(c *Consumer) {
		c.queueConfig.maxMessages = max
	}
}

// Sets the visibility timeout before SQS will requeue the message
// and send to other consumers
// Default: 60s
func WithVisibilityTimeout(visibility int) ConsumerOptions {
	return func(c *Consumer) {
		c.queueConfig.visibilityTimeout = visibility
	}
}

// Sets optional attributes to get returned from the message
func WithQueueAttributeNames(names []string) ConsumerOptions {
	return func(c *Consumer) {
		c.queueConfig.queueAttributeNames = names
	}
}

// Sets how long (in seconds) the consumer will sit idle when there
// are no messages in the queue.  If a messages appears
// in the queue while idle, it will be received immediatly
func WithMaxIdleTime(max int) ConsumerOptions {
	return func(c *Consumer) {
		c.queueConfig.maxIdleTime = max
	}
}

func WithDispatchStrategy(strategy runner.DispatchStrategy) ConsumerOptions {
	return func(c *Consumer) {
		c.dispatchStrategy = strategy
	}
}

func WithAckBatchSize(size int) ConsumerOptions {
	return func(c *Consumer) {
		c.batchSize = size
	}
}

func WithAckBatchInterval(interval time.Duration) ConsumerOptions {
	return func(c *Consumer) {
		c.batchInterval = interval
	}
}

// Overrides how the AWS sdk will resolve the SQS endpoint
func WithEndpointResolver(resolver sqs.EndpointResolverV2) ConsumerOptions {
	return func(c *Consumer) {
		c.endpointResolver = resolver
	}
}

// Overrides how the AWS sdk gets the credientials
func WithCredentialProvider(resolver aws.CredentialsProvider) ConsumerOptions {
	return func(c *Consumer) {
		c.credentialProvider = resolver
	}
}
