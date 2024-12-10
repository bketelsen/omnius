package modules

import (
	"log/slog"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/delaneyj/toolbelt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Module interface {
	Poll() toolbelt.CtxErrFunc
	CreateStore(stores *stores.KVStores) error
}

type BaseModule struct {
	NatsClient *nats.Conn
	JetStream  jetstream.JetStream
	Logger     *slog.Logger
	Store      jetstream.KeyValue
}
