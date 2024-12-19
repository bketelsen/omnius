package incus

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/lxc/incus/v6/shared/api"
	"github.com/zeebo/xxh3"
)

func (d *IncusModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Incus Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping incus updates")
			return

		case <-time.After(Interval):
			d.Logger.Debug("tick")

			// containers
			var (
				instances []api.Instance
				images    []api.Image
				err       error
			)
			if instances, err = d.client.GetInstancesAllProjects(api.InstanceTypeAny); err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			b, err := json.Marshal(instances)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			// hash the data
			h := hash(b)
			// get the current hash

			currentVal, err := d.Store.Get(context.Background(), "instances")
			if err != nil {
				d.Logger.Error(err.Error())
				if strings.Contains(err.Error(), "not found") {
					currentVal = nil
				}
			}
			if currentVal != nil {
				if h != hash(currentVal.Value()) {
					// update
					d.Logger.Info("instances different, updating")
					if _, err := d.Store.Put(context.Background(), "instances", b); err != nil {
						d.Logger.Error(err.Error())

					}
				}
			} else {
				// no current value, set it
				d.Logger.Info("setting instances value")
				if _, err := d.Store.Put(context.Background(), "instances", b); err != nil {
					d.Logger.Error(err.Error())
					continue
				}
			}
			if images, err = d.client.GetImagesAllProjects(); err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			b, err = json.Marshal(images)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			// hash the data
			ih := hash(b)
			// get the current hash

			currentVal, err = d.Store.Get(context.Background(), "images")
			if err != nil {
				d.Logger.Error(err.Error())
				if strings.Contains(err.Error(), "not found") {
					currentVal = nil
				}
			}
			if currentVal != nil {
				if ih != hash(currentVal.Value()) {
					// update
					d.Logger.Info("images different, updating")
					if _, err := d.Store.Put(context.Background(), "images", b); err != nil {
						d.Logger.Error(err.Error())

					}
				}
			} else {
				// no current value, set it
				d.Logger.Info("setting images value")
				if _, err := d.Store.Put(context.Background(), "images", b); err != nil {
					d.Logger.Error(err.Error())
					continue
				}
			}

			// create a status for the overview/system page
			ds := IncusStatus{
				ActiveInstances: strconv.Itoa(len(instances)),
				ImageCount:      strconv.Itoa(len(images)),
			}
			b, err = json.Marshal(ds)
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			if _, err := d.Store.Put(context.Background(), "incus_status", b); err != nil {
				d.Logger.Error(err.Error())
				continue
			}

		}
	}

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
