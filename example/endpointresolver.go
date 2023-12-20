package main

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithyEndpoints "github.com/aws/smithy-go/endpoints"
)

type SqsLocalConfig struct {
	Endpoint string
}

func (s *SqsLocalConfig) ResolveEndpoint(ctx context.Context, params sqs.EndpointParameters) (
	smithyEndpoints.Endpoint, error,
) {
	endpoint, err := url.Parse(s.Endpoint)
	if err != nil {
		return smithyEndpoints.Endpoint{}, err
	}

	return smithyEndpoints.Endpoint{
		URI: *endpoint,
	}, err
}
