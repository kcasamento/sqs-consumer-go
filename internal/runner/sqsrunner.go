package runner

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/kcasamento/sqs-consumer-go/internal/service"
	"github.com/kcasamento/sqs-consumer-go/internal/worker"
	"github.com/kcasamento/sqs-consumer-go/types"
)

type DispatchStrategy int

const (
	SemPool DispatchStrategy = iota + 1
	WorkerPool
)

type SqsRunner struct {
	stop                chan struct{}
	worker              worker.Worker
	handler             types.HandleMessage[awstypes.Message]
	client              service.Sqs
	queueUrl            string
	queueAttributeNames []string
	maxIdleTime         int
	concurrency         int
	visibilityTimeout   int
	maxMessages         int
	dispatchStrategy    DispatchStrategy
}

func NewSqsRunner(
	handler types.HandleMessage[awstypes.Message],
	batchSize int,
	batchInterval time.Duration,
	client service.Sqs,
	queueUrl string,
	concurrency int,
	visibilityTimeout int,
	maxMessages int,
	queueAttributeNames []string,
	maxIdleTime int,
	dispatchStrategy DispatchStrategy,
) Runner {
	r := &SqsRunner{
		maxIdleTime:         maxIdleTime,
		concurrency:         concurrency,
		visibilityTimeout:   visibilityTimeout,
		maxMessages:         maxMessages,
		stop:                make(chan struct{}, 1),
		handler:             handler,
		client:              client,
		queueUrl:            queueUrl,
		queueAttributeNames: queueAttributeNames,
		dispatchStrategy:    dispatchStrategy,
	}

	dispatcher := worker.WithSemPool()
	if dispatchStrategy == WorkerPool {
		dispatcher = worker.WithWorkerPool()
	}

	w := worker.NewSqsWorker(
		handler,
		r.ack,
		concurrency,
		dispatcher,
		worker.WithMaxBatchSize(batchSize),
		worker.WithBatchFlushInterval(batchInterval),
	)

	r.worker = w

	return r
}

func (r *SqsRunner) Run(ctx context.Context) {
	go r.run(ctx)
}

func (w *SqsRunner) Stop(ctx context.Context) {
	w.stop <- struct{}{}

	w.worker.Stop(ctx)
}

func (w *SqsRunner) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// TODO: metric
			return
		case <-w.stop:
			// TODO: metric
			return
		default:
			w.tick(ctx)
		}
	}
}

func (r *SqsRunner) tick(ctx context.Context) {
	receiveMessageInput := &sqs.ReceiveMessageInput{
		QueueUrl:              &r.queueUrl,
		MessageAttributeNames: r.queueAttributeNames,
		MaxNumberOfMessages:   int32(r.maxMessages),
		VisibilityTimeout:     int32(r.visibilityTimeout),
		WaitTimeSeconds:       int32(r.maxIdleTime),
	}

	msgResult, err := r.client.ReceiveMessage(ctx, receiveMessageInput)
	if err != nil {
		// TODO: metric
		log.Printf("error receiving message: %v\n", err)
		return
	}

	if err := r.worker.Submit(ctx, msgResult); err != nil {
		log.Printf("error submitting message: %v\n", err)
	}
}

func (r *SqsRunner) ack(messages []awstypes.Message) {
	ctx := context.Background()

	batch := make([]awstypes.DeleteMessageBatchRequestEntry, len(messages))
	for i, msg := range messages {
		batch[i] = awstypes.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		}
	}
	_, err := r.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		Entries:  batch,
		QueueUrl: &r.queueUrl,
	})
	if err != nil {
		log.Printf("error deleting batch of messages: %v\n", err)
	}
}
