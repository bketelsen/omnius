package incus

import (
	"context"
	"time"
)

func (d *IncusModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Incus Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping incus updates")
			return

		case <-time.After(Interval):
			d.Logger.Info("tick")

		}
	}

}
