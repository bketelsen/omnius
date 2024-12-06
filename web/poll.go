package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func poll(ctx context.Context, ns *embeddednats.Server, stores *stores.KVStores) error {

	nc, err := ns.Client()
	if err != nil {
		return fmt.Errorf("error creating nats client: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("error creating jetstream client: %w", err)
	}

	dockerkv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      "docker",
		Description: "Omnius Docker Key Value Store",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		return fmt.Errorf("error creating key value: %w", err)
	}
	stores.DockerStore = dockerkv

	systemkv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      "system",
		Description: "Omnius System Key Value Store",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		return fmt.Errorf("error creating key value: %w", err)
	}
	stores.SystemStore = systemkv

	egctx := toolbelt.NewErrGroupSharedCtx(ctx,
		pollDocker(ctx, stores.DockerStore),
		pollSystem(ctx, stores.SystemStore),
	)
	return egctx.Wait()

}

type CPUSimple struct {
	UsedPercent string `json:"usedPercent"`
	Used        string `json:"used"`
	Cores       int    `json:"cores"`
}

func pollSystem(ctx context.Context, systemkv jetstream.KeyValue) toolbelt.CtxErrFunc {
	return func(ctxp context.Context) (err error) {
		for {
			select {
			case <-ctx.Done():
				defer slog.Info("Stopping system updates")
				return
			case <-time.After(2 * time.Second):
				slog.Info("system tick")
				var (
					err error
				)
				v, err := mem.VirtualMemory()

				if err != nil {
					return fmt.Errorf("error getting memory: %w", err)
				}
				b, err := json.Marshal(v)
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				if _, err := systemkv.Put(context.Background(), "virtualMemory", b); err != nil {
					slog.Error(err.Error())

					continue
				}
				// cpu
				cores, err := cpu.Counts(true)
				if err != nil {
					return fmt.Errorf("error getting cpu counts: %w", err)
				}

				usage, err := cpu.Percent(0, false)
				if err != nil {
					return fmt.Errorf("error getting cpu percent: %w", err)
				}
				used := fmt.Sprintf("%.2f", usage[0])
				b, err = json.Marshal(CPUSimple{
					UsedPercent: used,
					Used:        fmt.Sprintf("%.0f", usage[0]),
					Cores:       cores,
				})
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				if _, err := systemkv.Put(context.Background(), "cpu", b); err != nil {
					slog.Error(err.Error())

					continue
				}

			}
		}

	}
}

func pollDocker(ctx context.Context, dockerkv jetstream.KeyValue) toolbelt.CtxErrFunc {

	return func(ctxp context.Context) (err error) {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("error creating docker connection: %w", err)
		}
		defer cli.Close()
		for {
			select {
			case <-ctx.Done():
				defer slog.Info("Stopping docker updates")
				return
			case <-time.After(5 * time.Second):
				slog.Info("docker image tick")

				// images
				var (
					images []image.Summary
				)
				if images, err = cli.ImageList(context.Background(), image.ListOptions{}); err != nil {
					slog.Error(err.Error())
					continue
				}
				b, err := json.Marshal(images)
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				if _, err := dockerkv.Put(context.Background(), "images", b); err != nil {
					slog.Error(err.Error())

					continue
				}
			case <-time.After(1 * time.Second):
				slog.Info("docker container tick")

				// containers
				var (
					containers []types.Container
					err        error
				)
				if containers, err = cli.ContainerList(context.Background(), containertypes.ListOptions{}); err != nil {
					slog.Error(err.Error())
					continue
				}
				b, err := json.Marshal(containers)
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				if _, err := dockerkv.Put(context.Background(), "containers", b); err != nil {
					slog.Error(err.Error())

					continue
				}

			}
		}

	}
}
