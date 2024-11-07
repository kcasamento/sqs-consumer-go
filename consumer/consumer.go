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

	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
	handler            types.HandleMessage[awstypes.Message]
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

func New(
	queueUrl string,
	handler types.HandleMessage[awstypes.Message],
	opts ...ConsumerOptions,
) (*Consumer, func(context.Context) error, error) {
	// default empty config
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

	// range over options
	for _, option := range opts {
		option(consumer)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, nil, err
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
	if consumer.client == nil {
		// set the client to interact with sqs
		consumer.client = sqsClient
	}

	// setup heartbeat
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

	return consumer, consumer.Stop, nil
}

func (c *Consumer) Start(ctx context.Context) {
	if c.heartbeat != nil {
		// starts the heartbeat service in the
		// background
		c.heartbeat.Start(ctx)
	}

	// open the flood gates
	// and start processing messages
	c.runner.Run(ctx)
}

func (c *Consumer) Stop(ctx context.Context) error {
	if c.heartbeat != nil {
		c.heartbeat.Stop(ctx)
	}
	c.runner.Stop(ctx)
	return nil
}
