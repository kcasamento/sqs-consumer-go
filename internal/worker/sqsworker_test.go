package worker

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
)

func BenchmarkSqsWorker_SemPool(b *testing.B) {
	w := NewSqsWorker(
		func(context.Context, string, awstypes.Message) (bool, error) {
			return false, nil
		},
		func(_ []awstypes.Message) {},
		100,
		WithSemPool(),
	)

	for n := 0; n < b.N; n++ {
		val := fmt.Sprintf("%d", n)
		w.Submit(context.Background(), &sqs.ReceiveMessageOutput{
			Messages: []awstypes.Message{
				{
					Body:          &val,
					MessageId:     &val,
					ReceiptHandle: &val,
				},
			},
			ResultMetadata: middleware.Metadata{},
		})
	}
}

func BenchmarkSqsWorker_WorkerPool(b *testing.B) {
	w := NewSqsWorker(
		func(context.Context, string, awstypes.Message) (bool, error) {
			return false, nil
		},
		func([]awstypes.Message) {},
		100,
		WithWorkerPool(),
	)

	for n := 0; n < b.N; n++ {
		val := fmt.Sprintf("%d", n)
		w.Submit(context.Background(), &sqs.ReceiveMessageOutput{
			Messages: []awstypes.Message{
				{
					Body:          &val,
					MessageId:     &val,
					ReceiptHandle: &val,
				},
			},
			ResultMetadata: middleware.Metadata{},
		})
	}
}
