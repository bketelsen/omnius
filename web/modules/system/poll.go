package system

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func (d *SystemModule) Poll(ctx context.Context) {
	d.Logger.Info("Polling System Module")

	systemdConnection, err := dbus.NewSystemConnectionContext(context.Background())
	if err != nil {
		d.Logger.Error("systemd dbus connection", "error", err)
	}
	defer systemdConnection.Close()
	for {
		select {
		case <-ctx.Done():
			defer d.Logger.Info("Stopping system updates")
			return
		case <-time.After(2 * time.Second):
			d.Logger.Info("system tick")
			var (
				err error
			)
			v, err := mem.VirtualMemory()

			if err != nil {
				d.Logger.Error("error getting memory", "error", err)
				continue
			}
			b, err := json.Marshal(v)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			if _, err := d.Store.Put(context.Background(), "virtualMemory", b); err != nil {
				d.Logger.Error(err.Error())

				continue
			}
			// cpu
			cores, err := cpu.Counts(true)
			if err != nil {
				d.Logger.Error("error getting cpu counts", "error", err)
				continue
			}

			usage, err := cpu.Percent(0, false)
			if err != nil {
				d.Logger.Error("error getting cpu percent", "error", err)
				continue
			}
			used := fmt.Sprintf("%.2f", usage[0])
			b, err = json.Marshal(CPUSimple{
				UsedPercent: used,
				Used:        fmt.Sprintf("%.0f", usage[0]),
				Cores:       cores,
			})
			if err != nil {
				d.Logger.Error(err.Error())
				continue
			}
			if _, err := d.Store.Put(context.Background(), "cpu", b); err != nil {
				d.Logger.Error(err.Error())

				continue
			}

			// systemd units
			if systemdConnection != nil {
				units, err := systemdConnection.ListUnitsByPatternsContext(context.Background(), []string{"running"}, []string{"*.service"})
				if err != nil {
					d.Logger.Error("error getting systemd services", "error", err)
					continue
				}
				b, err = json.Marshal(units)
				if err != nil {
					d.Logger.Error(err.Error())
					continue
				}
				if _, err := d.Store.Put(context.Background(), "services", b); err != nil {
					d.Logger.Error(err.Error())

					continue
				}
			}

		}
	}

}
