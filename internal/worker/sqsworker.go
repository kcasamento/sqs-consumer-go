package worker

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/kcasamento/sqs-consumer-go/internal/harvester"
	"github.com/kcasamento/sqs-consumer-go/types"
)

type (
	Ack        func([]awstypes.Message)
	Dispatcher interface {
		Dispatch(ctx context.Context, task func()) error
	}
	SqsWorkerOpt func(*SqsWorker)
)

func WithSemPool() SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.wPool = NewSemPool(w.concurrency, 10*time.Second)
	}
}

func WithWorkerPool() SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.wPool = NewWorker(w.concurrency)
	}
}

func WithMaxBatchSize(maxBatchSize int) SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.maxBatchSize = maxBatchSize
	}
}

func WithBatchFlushInterval(batchFlushInterval time.Duration) SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.batchFlushInterval = batchFlushInterval
	}
}

var _ Worker[*sqs.ReceiveMessageOutput] = (*SqsWorker)(nil)

type SqsWorker struct {
	stop               chan struct{}
	wPool              Dispatcher
	handler            types.HandleMessage[awstypes.Message]
	harvester          harvester.Harvester[awstypes.Message]
	batchFlushInterval time.Duration
	maxBatchSize       int
	concurrency        int
}

func NewSqsWorker(
	handler types.HandleMessage[awstypes.Message],
	ack Ack,
	concurrency int,
	opts ...SqsWorkerOpt,
) *SqsWorker {
	w := &SqsWorker{
		concurrency:        concurrency,
		stop:               make(chan struct{}, 1),
		handler:            handler,
		maxBatchSize:       10,
		batchFlushInterval: 5 * time.Second,
		wPool:              NewSemPool(concurrency, 10*time.Second),
	}

	for _, opt := range opts {
		opt(w)
	}

	w.harvester = harvester.NewBatchHarvester[awstypes.Message](ack, w.maxBatchSize, w.batchFlushInterval)

	return w
}

func (w *SqsWorker) Submit(ctx context.Context, message *sqs.ReceiveMessageOutput) error {
	// Now that we know we have the sqs objects, range over
	// the individual messages in the batch, and dispatch
	// them to the worker pool
	for _, msg := range message.Messages {
		_ = w.wPool.Dispatch(ctx, func() {
			w.handleMessage(ctx, msg)
		})
	}

	return nil
}

func (w *SqsWorker) Stop(ctx context.Context) error {
	w.harvester.Stop()
	return nil
}

func (w *SqsWorker) handleMessage(ctx context.Context, message awstypes.Message) {
	// TODO: generating uuids will be inefficient
	// under heavy load...currently for demo purposes
	processId := uuid.New().String()

	// TODO: metric
	// log.Printf("process %s started", processId)

	retry, err := w.handler(ctx, processId, message)
	if err != nil {
		// TODO: metric
		log.Printf("error from handler: %v, retry=%t\n", err, retry)

		if retry {
			// since we ack at the end of the function
			// returning here will skip the ack
			// and keep the message on the queue
			return
		}
		// TODO: metric
	}

	// the message was processed but we
	// won't necessarily ack it right away
	// but rather queue it up in memory
	// and flush the messages in batches
	// using the harvester
	w.harvester.Add(message)
}
