package stores

import "github.com/nats-io/nats.go/jetstream"

type KVStores struct {
	IncusStore   jetstream.KeyValue
	DockerStore  jetstream.KeyValue
	SystemStore  jetstream.KeyValue
	StorageStore jetstream.KeyValue
}
