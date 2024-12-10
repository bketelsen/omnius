package docker

import (
	"log/slog"
	"time"

	"github.com/bketelsen/omnius/web/modules"
	"github.com/bketelsen/omnius/web/stores"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zeebo/xxh3"
)

const (
	ModuleName        = "docker"
	Group             = "containers"
	BucketName        = "docker"
	BucketDescription = "Docker Container and Image Information"
	Interval          = 1 * time.Second
)

func init() {
	// automatically register this module as available to initialize
	modules.Register(ModuleName, &DockerModule{})

}

// ensure we implement the Module interface
var _ modules.Module = &DockerModule{}

type DockerModule struct {
	modules.BaseModule
	client    *client.Client
	hasDocker bool
}

func (d *DockerModule) Init(logger *slog.Logger, stores *stores.KVStores, nc *nats.Conn, js jetstream.JetStream) error {

	d.Logger = logger.With("module", ModuleName)
	d.NatsClient = nc
	d.JetStream = js
	d.Stores = stores

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		d.hasDocker = false
	} else {
		d.client = cli
		d.hasDocker = true
		d.CreateStore(stores)

	}
	return nil
}

func (d *DockerModule) Enabled() bool {
	return d.hasDocker
}
func (d *DockerModule) Group() string {
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
