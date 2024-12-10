package storage

import (
	"context"
	"time"
)

func (d *StorageModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Storage Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping storage updates")
			return

		case <-time.After(Interval):
			d.Logger.Info("tick")

		}
	}

}
