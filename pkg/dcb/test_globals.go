package dcb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
)

var (
	ctx       context.Context
	pool      *pgxpool.Pool
	store     EventStore
	container testcontainers.Container
)
