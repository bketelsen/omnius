package modules

import (
	"context"
	"log/slog"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var AvailableModules = make(map[string]Module)

type Module interface {
	Init(*slog.Logger, *stores.KVStores, *nats.Conn, jetstream.JetStream) error
	Poll(context.Context)
	SetupRoutes(chi.Router, context.Context) error
	CreateStore(stores *stores.KVStores) error
	Enabled() bool
}

type BaseModule struct {
	NatsClient *nats.Conn
	JetStream  jetstream.JetStream
	Logger     *slog.Logger
	Store      jetstream.KeyValue
	Stores     *stores.KVStores
}

func Register(name string, m Module) {
	AvailableModules[name] = m
}
