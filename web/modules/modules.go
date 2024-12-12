package modules

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/bketelsen/omnius/web/components"
	"github.com/bketelsen/omnius/web/layouts"
	"github.com/bketelsen/omnius/web/stores"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var AvailableModules = make(map[string]Module)

type Module interface {
	// Init initializes the module with the logger, stores, nats connection, and jetstream
	Init(*slog.Logger, *stores.KVStores, *nats.Conn, jetstream.JetStream) error
	// Poll for this module's data and publish to the NATS server
	Poll(context.Context)
	// SetupRoutes creates the http routes for this module
	SetupRoutes(chi.Router, []*layouts.SidebarGroup, context.Context) error
	// CreateStore creates a key value store for this module
	CreateStore(stores *stores.KVStores) error
	// Enabled returns true if this module is enabled
	Enabled() bool
	// Group returns the group name for this module
	Group() string
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

type ModuleUpdate struct {
	Module string
}

func (b *BaseModule) CreateToast(t components.Toast) error {

	toastBytes, err := json.Marshal(t)
	if err != nil {
		b.Logger.Error(err.Error())
		return err
	} else {
		uuid := uuid.New().String()
		_, err = b.Stores.MessageStore.Put(context.Background(), uuid, toastBytes)
		if err != nil {
			b.Logger.Error(err.Error())

			return err
		}

		return nil
	}

}

// ExpireToast deletes the notification from the message store after `duration`
// has passed
func (b *BaseModule) ExpireToast(id string, duration time.Duration) {

	go func() {
		time.Sleep(duration)
		err := b.Stores.MessageStore.Delete(context.Background(), id)
		if err != nil {
			b.Logger.Error(err.Error())
		}
	}()

}
