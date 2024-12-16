package logs

import (
	"context"
	"time"
)

func (d *LogModule) Poll(ctx context.Context) {

	d.Logger.Info("Polling Log Module")

	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping log updates")
			return

		case <-time.After(Interval):
			d.Logger.Debug("tick")

		}
	}

}
