package runner

import "context"

type Runner interface {
	Run(context.Context)
	Stop(context.Context)
}
