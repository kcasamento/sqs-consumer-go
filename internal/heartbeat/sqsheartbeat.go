package heartbeat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/kcasamento/sqs-consumer-go/internal/service"
	ctypes "github.com/kcasamento/sqs-consumer-go/types"
)

const (
	HEARTBEAT_PID = "heartbeat"
)

type SqsHeartbeat struct {
	handler           ctypes.HandleMessage
	q                 chan string
	stop              chan struct{}
	client            service.Sqs
	queueUrl          string
	visibilityTimeout int
	heartbeatInterval time.Duration
}

func NewSqsHeartbeat(
	client service.Sqs,
	queueUrl string,
	heartbeatInterval time.Duration,
	visibilityTimeout int,
	bufferSize int,
	handler ctypes.HandleMessage,
) (ctypes.HandleMessage, Heartbeat) {
	h := &SqsHeartbeat{
		client:            client,
		queueUrl:          queueUrl,
		heartbeatInterval: heartbeatInterval,
		visibilityTimeout: visibilityTimeout,
		q:                 make(chan string, bufferSize),
		stop:              make(chan struct{}),
	}

	return h.wrapHandler(handler), h
}

func (h *SqsHeartbeat) Start(ctx context.Context) error {
	if h == nil {
		return nil
	}

	go h.ProcessHeartbeats(ctx)

	// TODO: metric

	return nil
}

func (h *SqsHeartbeat) Stop(_ context.Context) error {
	if h == nil {
		return nil
	}

	h.stop <- struct{}{}
	return nil
}

func (h *SqsHeartbeat) wrapHandler(handler ctypes.HandleMessage) ctypes.HandleMessage {
	return func(ctx context.Context, processId string, message interface{}) (bool, error) {
		msg, ok := message.(*types.Message)
		if !ok {
			return false, fmt.Errorf("invalid message type for sqs heartbeat")
		}

		ticker := time.NewTicker(h.heartbeatInterval)
		done := make(chan struct{})

		defer func() {
			done <- struct{}{}
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					h.SendHeatbeat(*msg.ReceiptHandle)
				}
			}
		}()

		return handler(ctx, processId, msg)
	}
}

func (h *SqsHeartbeat) SendHeatbeat(receiptHandle string) {
	log.Printf("sending heartbeat\n")
	h.q <- receiptHandle
}

func (h *SqsHeartbeat) ProcessHeartbeats(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// TODO: metric
			return
		case <-h.stop:
			// TODO: metric
			return
		case receiptHandle := <-h.q:
			h.updateVisibility(ctx, receiptHandle)
		}
	}
}

func (h *SqsHeartbeat) updateVisibility(ctx context.Context, receiptHandle string) error {
	_, err := h.client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &h.queueUrl,
		ReceiptHandle:     &receiptHandle,
		VisibilityTimeout: int32(h.visibilityTimeout),
	})
	if err != nil {
		if strings.Contains(err.Error(), "Message does not exist") {
			// Heatbeat was sent after messaged was ack'd so we can just
			// ignore this error
			return nil
		}
		// TODO: metric
		log.Printf("error updating message visibility: %v", err)
		return err
	}

	// TODO: metric
	log.Printf("updated message visibility for %s", receiptHandle)
	return nil
}
