package consumer

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/kcasamento/sqs-consumer-go/internal/heartbeat"
	"github.com/kcasamento/sqs-consumer-go/internal/runner"
	"github.com/kcasamento/sqs-consumer-go/internal/service"
	"github.com/kcasamento/sqs-consumer-go/types"
)

type heartbeatConfig struct {
	interval time.Duration
}

type queueConfig struct {
	queueUrl            string
	queueAttributeNames []string
	visibilityTimeout   int
	maxMessages         int
	maxIdleTime         int
}

type Consumer struct {
	runner             runner.Runner
	client             service.Sqs
	handler            types.HandleMessage
	heartbeatConfig    *heartbeatConfig
	queueConfig        *queueConfig
	endpointResolver   sqs.EndpointResolverV2
	credentialProvider aws.CredentialsProvider
	heartbeat          heartbeat.Heartbeat
	concurrency        int
	dispatchStrategy   runner.DispatchStrategy
	batchSize          int
	batchInterval      time.Duration
}

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

func New(
	queueUrl string,
	handler types.HandleMessage,
	opts ...ConsumerOptions,
) (*Consumer, error) {
	consumer := &Consumer{
		runner:          nil,
		client:          nil,
		concurrency:     100,
		handler:         handler,
		batchSize:       10,
		batchInterval:   5 * time.Second,
		heartbeatConfig: nil,
		heartbeat:       nil,
		queueConfig: &queueConfig{
			queueUrl:            queueUrl,
			visibilityTimeout:   60,
			maxMessages:         10,
			queueAttributeNames: []string{},
			maxIdleTime:         5,
		},
		endpointResolver:   nil,
		credentialProvider: nil,
		dispatchStrategy:   runner.SemPool,
	}

	for _, option := range opts {
		option(consumer)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	c := sqs.NewFromConfig(cfg, func(options *sqs.Options) {
		if consumer.endpointResolver != nil {
			options.EndpointResolverV2 = consumer.endpointResolver
		}

		if consumer.credentialProvider != nil {
			options.Credentials = consumer.credentialProvider
		}
	})

	sqsClient := service.NewSqsClient(c)

	consumer.client = sqsClient

	if consumer.heartbeatConfig != nil {
		hbHandler, hb := heartbeat.NewSqsHeartbeat(
			consumer.client,
			consumer.queueConfig.queueUrl,
			consumer.heartbeatConfig.interval,
			consumer.queueConfig.visibilityTimeout,
			consumer.concurrency,
			consumer.handler,
		)

		consumer.handler = hbHandler
		consumer.heartbeat = hb
	}

	r := runner.NewSqsRunner(
		consumer.handler,
		consumer.batchSize,
		consumer.batchInterval,
		consumer.client,
		consumer.queueConfig.queueUrl,
		consumer.concurrency,
		consumer.queueConfig.visibilityTimeout,
		consumer.queueConfig.maxMessages,
		consumer.queueConfig.queueAttributeNames,
		consumer.queueConfig.maxIdleTime,
		consumer.dispatchStrategy,
	)

	consumer.runner = r

	return consumer, nil
}

func (c *Consumer) Start(ctx context.Context) {
	if c.heartbeat != nil {
		c.heartbeat.Start(ctx)
	}
	c.runner.Run(ctx)
}

func (c *Consumer) Stop(ctx context.Context) {
	if c.heartbeat != nil {
		c.heartbeat.Stop(ctx)
	}
	c.runner.Stop(ctx)
}
