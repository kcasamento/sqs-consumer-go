package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/kcasamento/sqs-consumer-go/types"
	"golang.org/x/sync/semaphore"
)

type Ack func(context.Context, *awstypes.Message) error

type SqsWorker struct {
	stop        chan struct{}
	sem         *semaphore.Weighted
	handler     types.HandleMessage
	ack         Ack
	concurrency int
	// TODO: retry mode
}

func NewSqsWorker(
	handler types.HandleMessage,
	ack Ack,
	concurrency int,
) Worker {
	w := &SqsWorker{
		concurrency: concurrency,
		stop:        make(chan struct{}, 1),
		handler:     handler,
		sem:         semaphore.NewWeighted(int64(concurrency)),
		ack:         ack,
	}

	return w
}

func (w *SqsWorker) Submit(ctx context.Context, message interface{}) error {
	msgResult, ok := message.(*sqs.ReceiveMessageOutput)
	if !ok {
		return fmt.Errorf("invalid message type for sqs workers")
	}
	return w.controlledSemSubmit(ctx, msgResult)
}

func (w *SqsWorker) Stop(ctx context.Context) error {
	return nil
}

func (w *SqsWorker) controlledSemSubmit(ctx context.Context, msgResult *sqs.ReceiveMessageOutput) error {
	timeoutCtx, timeout := context.WithTimeout(ctx, 10*time.Second)
	defer timeout()

	h := func(ctx context.Context, message *awstypes.Message) {
		defer w.sem.Release(1)
		w.handleMessage(ctx, message)
	}

	for _, msg := range msgResult.Messages {
		if err := w.sem.Acquire(timeoutCtx, 1); err != nil {
			return err
		}
		go h(ctx, &msg)
	}

	return nil
}

func (w *SqsWorker) handleMessage(ctx context.Context, message *awstypes.Message) {
	defer w.sem.Release(1)

	processId := uuid.New().String()
	// TODO: metric
	log.Printf("process %s started", processId)

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
	log.Printf("message %s has been ack'd\n", *message.MessageId)
}
