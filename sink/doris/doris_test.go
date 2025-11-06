package doris

import (
	"context"
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestOutput_Start(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("should return error when client is nil", func(t *testing.T) {
		output := &DorisOutput{
			Host:     "10.99.1.6:9030",
			Database: "demo",
			Table:    "test_table",
		}
		err := output.Start(ctx, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer of doris output client")
	})

	t.Run("should start successfully with valid config", func(t *testing.T) {
		output := &DorisOutput{
			Host:              "10.99.1.6:8030",
			Database:          "demo",
			Table:             "test_table",
			BulkActions:       5 << 20,
			BulkFlushInterval: time.Second * 10,
			client:            &Client{}, // Mock client for test
		}

		err := output.Start(ctx, logger)
		assert.NoError(t, err)
		assert.NotNil(t, output.processor)
		assert.Equal(t, logger, output.logger)
	})

	t.Run("should process messages after start", func(t *testing.T) {
		output := &DorisOutput{
			Host:              "10.99.1.6:8030",
			Database:          "test",
			Table:             "test_table",
			BulkActions:       1024,
			BulkFlushInterval: time.Second,
			client:            &Client{}, // Mock client for test
		}

		err := output.Start(ctx, logger)
		assert.NoError(t, err)

		// Test sending a message
		testLog := types.ExtractedLog{"field": "value"}
		err = output.Sink(ctx, testLog)
		assert.NoError(t, err)

		// Clean up
		output.Stop(ctx)
	})
}

func TestOut_NewClient(t *testing.T) {
	out := DorisOutput{
		Host:     "10.99.1.6:9030",
		Database: "lfn_test",
		Username: "root",
		Password: "",
		Table:    "test",
	}
	if cli, err := out.NewClient(context.Background()); err != nil {
		t.Fatal(err)
	} else {
		t.Log(cli)
	}
}
