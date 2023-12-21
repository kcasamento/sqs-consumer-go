package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/kcasamento/sqs-consumer-go/types"
)

type (
	Ack        func(context.Context, *awstypes.Message) error
	Dispatcher interface {
		Dispatch(ctx context.Context, task func()) error
	}
	SqsWorkerOpt func(*SqsWorker)
)

func WithSemPool() SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.wPool = NewSemPool(w.concurrency)
	}
}

func WithWorkerPool() SqsWorkerOpt {
	return func(w *SqsWorker) {
		w.wPool = NewWorker(w.concurrency)
	}
}

type SqsWorker struct {
	stop        chan struct{}
	wPool       Dispatcher
	handler     types.HandleMessage
	ack         Ack
	concurrency int
}

func NewSqsWorker(
	handler types.HandleMessage,
	ack Ack,
	concurrency int,
	opts ...SqsWorkerOpt,
) Worker {
	w := &SqsWorker{
		concurrency: concurrency,
		stop:        make(chan struct{}, 1),
		handler:     handler,
		ack:         ack,
		wPool:       NewSemPool(concurrency),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

func (w *SqsWorker) Submit(ctx context.Context, message interface{}) error {
	msgResult, ok := message.(*sqs.ReceiveMessageOutput)
	if !ok {
		return fmt.Errorf("invalid message type for sqs workers")
	}

	for _, msg := range msgResult.Messages {
		_ = w.wPool.Dispatch(ctx, func() {
			w.handleMessage(ctx, &msg)
		})
	}

	return nil
}

func (w *SqsWorker) Stop(ctx context.Context) error {
	return nil
}

func (w *SqsWorker) handleMessage(ctx context.Context, message *awstypes.Message) {
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

	if err := w.ack(ctx, message); err != nil {
		// Error ack'ing message...
		// TODO: metric
		log.Printf("error acking message: %v\n", err)
		return
	}

	// handle message succeeded
	// TODO: metric
	// log.Printf("message %s has been ack'd\n", *message.MessageId)
}
