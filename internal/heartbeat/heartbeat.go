package heartbeat

import (
	"context"
)

type (
	HeartbeatFunc func(ctx context.Context, message any)
	Heartbeat     interface {
		Start(context.Context) error
		Stop(context.Context) error
	}
)
