package types

import (
	"context"
)

type HandleMessage func(context.Context, string, interface{}) (bool, error)
