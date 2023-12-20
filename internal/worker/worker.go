package worker

import "context"

type Worker interface {
	Submit(context.Context, interface{}) error
	Stop(context.Context) error
}
