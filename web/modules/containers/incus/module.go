package incus

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
	ModuleName        = "incus"
	Group             = "containers"
	BucketName        = "incus"
	BucketDescription = "Incus Container and Image Information"
	Interval          = 1 * time.Second
)

func init() {
	// automatically register this module as available to initialize
	modules.Register(ModuleName, &IncusModule{})

}

// ensure we implement the Module interface
var _ modules.Module = &IncusModule{}

type IncusModule struct {
	modules.BaseModule
}

func (d *IncusModule) Init(logger *slog.Logger, stores *stores.KVStores, nc *nats.Conn, js jetstream.JetStream) error {

	d.Logger = logger.With("module", ModuleName)
	d.NatsClient = nc
	d.JetStream = js
	d.Stores = stores

	d.CreateStore(stores)

	return nil
}

func (d *IncusModule) Enabled() bool {
	return true
}
func (d *IncusModule) Group() string {
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
