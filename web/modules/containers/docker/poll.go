package docker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

func (d *DockerModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Docker Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping docker updates")
			return

		case <-time.After(Interval):
			d.Logger.Debug("tick")

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
					d.Logger.Debug("containers different, updating")
					if _, err := d.Store.Put(context.Background(), "containers", b); err != nil {
						slog.Error(err.Error())

					}
				}
			} else {
				// no current value, set it
				d.Logger.Debug("setting containers value")
				if _, err := d.Store.Put(context.Background(), "containers", b); err != nil {
					slog.Error(err.Error())
					continue
				}
			}
			// images
			var (
				images []image.Summary
			)
			if images, err = d.client.ImageList(context.Background(), image.ListOptions{
				Manifests:  true,
				SharedSize: true,
			}); err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			b, err = json.Marshal(images)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			d.Logger.Debug("setting images")
			if _, err := d.Store.Put(context.Background(), "images", b); err != nil {
				d.Logger.Error(err.Error())
				continue
			}

		}
	}

}
