#!/usr/bin/env bash

set -euo pipefail

echo "sending message..."
echo "==================="
LOCALSTACK_HOST=localhost
AWS_REGION=us-east-1



for i in {1..10}
do
  # aws --endpoint-url=http://${LOCALSTACK_HOST}:4566 sqs send-message --queue-url http://localstack:4566/000000000000/consumer-test-queue  --region ${AWS_REGION} --message-body '{
  #         "event_id": "7456c8ee-949d-4100-a0c6-6ae8e581ae15",
  #         "event_time": "2019-11-26T16:00:47Z",
  #         "data": {
  #           "test": $i 
  #       }
  #     }' > /dev/null
  aws --endpoint-url=http://${LOCALSTACK_HOST}:4566 sqs send-message --queue-url http://localstack:4566/000000000000/consumer-test-queue  --region ${AWS_REGION} --message-body "message $i" > /dev/null
done
