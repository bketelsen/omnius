package storage

import (
	"log/slog"
	"time"

	"github.com/bketelsen/omnius/web/modules"
	"github.com/bketelsen/omnius/web/stores"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zeebo/xxh3"
)

const (
	ModuleName        = "storage"
	Group             = "system"
	BucketName        = "storage"
	BucketDescription = "Storage Information"
	Interval          = 10 * time.Second
)

func init() {
	// automatically register this module as available to initialize
	modules.Register(ModuleName, &StorageModule{})

}

// ensure we implement the Module interface
var _ modules.Module = &StorageModule{}

type StorageModule struct {
	modules.BaseModule
}

func (d *StorageModule) Init(logger *slog.Logger, stores *stores.KVStores, nc *nats.Conn, js jetstream.JetStream) error {

	d.Logger = logger.With("module", ModuleName)
	d.NatsClient = nc
	d.JetStream = js
	d.Stores = stores

	d.CreateStore(stores)

	return nil
}

func (d *StorageModule) Enabled() bool {
	return true
}
func (d *StorageModule) Group() string {
	return Group
}

func hash(b []byte) uint64 {
	hasher := xxh3.New()
	defer hasher.Reset()

	_, err := hasher.Write(b)
	if err != nil {
		slog.Error(err.Error())
	}
	return hasher.Sum64()
}
