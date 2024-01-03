package harvester

type Harvester[T any] interface {
	Add(item T)
	Flush()
	Stop()
}
