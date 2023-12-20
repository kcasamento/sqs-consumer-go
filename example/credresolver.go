package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type LocalCredConfig struct{}

func (c *LocalCredConfig) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     "dummyId",
		SecretAccessKey: "dummyKey",
		SessionToken:    "sessionToken",
		Source:          "",
		CanExpire:       false,
	}, nil
}
