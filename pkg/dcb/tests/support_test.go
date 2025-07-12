package dcb

import (
	"context"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventStore TargetEventsTable validation", func() {
	It("should return a TableStructureError if the target table does not exist", func() {
		ctx := context.Background()
		config := dcb.EventStoreConfig{
			TargetEventsTable:      "nonexistent_events",
			MaxBatchSize:           1000,
			LockTimeout:            5000,
			StreamBuffer:           1000,
			DefaultAppendIsolation: 0,
			QueryTimeout:           15000,
			AppendTimeout:          10000,
		}
		_, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("table nonexistent_events does not exist"))
	})

	It("should return a TableStructureError if the target table has the wrong structure (missing column)", func() {
		ctx := context.Background()
		// Create a table with a missing 'occurred_at' column
		_, err := pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS bad_events (
				type VARCHAR(64) NOT NULL,
				tags TEXT[] NOT NULL,
				data JSON NOT NULL,
				transaction_id xid8 NOT NULL,
				position BIGSERIAL NOT NULL PRIMARY KEY
			);
		`)
		Expect(err).NotTo(HaveOccurred())

		config := dcb.EventStoreConfig{
			TargetEventsTable:      "bad_events",
			MaxBatchSize:           1000,
			LockTimeout:            5000,
			StreamBuffer:           1000,
			DefaultAppendIsolation: 0,
			QueryTimeout:           15000,
			AppendTimeout:          10000,
		}
		_, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing required column 'occurred_at'"))
	})
})
