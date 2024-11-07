package types

import (
	"context"
)

type HandleMessage[T interface{}] func(context.Context, string, T) (bool, error)
