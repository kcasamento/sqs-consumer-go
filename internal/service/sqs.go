package service

//go:generate mockgen --build_flags=--mod=mod -destination=mocks/sqs_mock.go -package=mocks . Sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Sqs interface {
	ChangeMessageVisibility(context.Context, *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessageBatch(context.Context, *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error)
}

type SqsClient struct {
	client *sqs.Client
}

func NewSqsClient(client *sqs.Client) *SqsClient {
	return &SqsClient{
		client: client,
	}
}

func (s *SqsClient) ChangeMessageVisibility(ctx context.Context, input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	return s.client.ChangeMessageVisibility(ctx, input)
}

func (s *SqsClient) ReceiveMessage(ctx context.Context, input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return s.client.ReceiveMessage(ctx, input)
}

func (s *SqsClient) DeleteMessageBatch(ctx context.Context, input *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	return s.client.DeleteMessageBatch(ctx, input)
}
