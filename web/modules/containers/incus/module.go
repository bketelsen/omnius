package incus

import (
	"log/slog"
	"time"

	"github.com/bketelsen/omnius/web/components"
	"github.com/bketelsen/omnius/web/modules"
	"github.com/bketelsen/omnius/web/stores"
	incus "github.com/lxc/incus/v6/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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
	client incus.InstanceServer
}

func (d *IncusModule) Init(logger *slog.Logger, stores *stores.KVStores, nc *nats.Conn, js jetstream.JetStream) error {

	d.Logger = logger.With("module", ModuleName)
	d.NatsClient = nc
	d.JetStream = js
	d.Stores = stores

	// Connect to Incus over the Unix socket
	c, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		errt := d.BaseModule.CreateToast(components.Toast{
			Message: "Unable to connect to incus",
			Type:    components.AlertError,
		})
		if errt != nil {
			d.Logger.Error("creating toast", "error", errt)
		}
		return err
	}
	d.client = c

	d.CreateStore(stores)

	return nil
}

func (d *IncusModule) Enabled() bool {
	return d.client != nil
}
func (d *IncusModule) Group() string {
	return Group
}
