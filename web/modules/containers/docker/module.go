package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bketelsen/omnius/web/modules"
	"github.com/bketelsen/omnius/web/stores"
	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zeebo/xxh3"
)

const (
	BucketName        = "docker"
	BucketDescription = "Docker Container and Image Information"
	Interval          = 1 * time.Second
)

func init() {

	modules.Register("docker", &DockerModule{})

}

// ensure we implement the Module interface
var _ modules.Module = &DockerModule{}

type DockerModule struct {
	modules.BaseModule
	client    *client.Client
	hasDocker bool
}

func (d *DockerModule) Init(logger *slog.Logger, stores *stores.KVStores, nc *nats.Conn, js jetstream.JetStream) error {

	d.Logger = logger.With("module", "docker")
	d.NatsClient = nc
	d.JetStream = js

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

func (d *DockerModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Docker Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping docker updates")
			return

		case <-time.After(Interval):
			d.Logger.Info("tick")

			// containers
			var (
				containers []types.Container
				err        error
			)
			if containers, err = d.client.ContainerList(context.Background(), containertypes.ListOptions{}); err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			b, err := json.Marshal(containers)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			// hash the data
			h := hash(b)
			// get the current hash

			currentVal, err := d.Store.Get(context.Background(), "containers")
			if err != nil {
				d.Logger.Error(err.Error())
				if strings.Contains(err.Error(), "not found") {
					currentVal = nil
				}
			}
			if currentVal != nil {
				if h != hash(currentVal.Value()) {
					// update
					d.Logger.Info("containers different, updating")
					if _, err := d.Store.Put(context.Background(), "containers", b); err != nil {
						slog.Error(err.Error())

					}
				}
			} else {
				// no current value, set it
				d.Logger.Info("setting containers value")
				if _, err := d.Store.Put(context.Background(), "containers", b); err != nil {
					slog.Error(err.Error())
					continue
				}
			}
			// images
			var (
				images []image.Summary
			)
			if images, err = d.client.ImageList(context.Background(), image.ListOptions{}); err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			b, err = json.Marshal(images)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			if _, err := d.Store.Put(context.Background(), "images", b); err != nil {
				d.Logger.Error(err.Error())
				continue
			}

		}
	}

}

func (d *DockerModule) CreateStore(stores *stores.KVStores) error {

	dockerkv, err := d.JetStream.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      BucketName,
		Description: BucketDescription,
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		d.hasDocker = false
		d.Logger.Error(err.Error())
		return fmt.Errorf("error creating key value: %w", err)
	}
	stores.DockerStore = dockerkv
	d.Store = dockerkv
	return nil
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
