package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/nats-io/nats.go/jetstream"
)

func (d *LogModule) CreateStore(stores *stores.KVStores) error {

	logkv, err := d.JetStream.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      BucketName,
		Description: BucketDescription,
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {

		d.Logger.Error(err.Error())
		return fmt.Errorf("error creating key value: %w", err)
	}
	stores.LogStore = logkv
	d.Store = logkv
	return nil
}
