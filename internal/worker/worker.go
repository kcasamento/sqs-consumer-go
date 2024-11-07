package worker

import (
	"context"
)

type Worker[T interface{}] interface {
	Submit(context.Context, T) error
	Stop(context.Context) error
}
