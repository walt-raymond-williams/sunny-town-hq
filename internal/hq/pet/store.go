package pet

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StarRewardRequest struct {
	EventID       string
	AppUserID     int64
	Source        string
	Delta         int
	RoomID        string
	MapID         string
	CollectibleID string
}

type InventoryConsumer func(ctx context.Context, tx pgx.Tx, userID int64, itemKey string, quantity int) (bool, error)

type StarRewardCommitter func(ctx context.Context, tx pgx.Tx, request StarRewardRequest) (bool, int, error)

type Store struct {
	DB                    *pgxpool.Pool
	CookieInventoryKey    string
	NoCookiesError        error
	ConsumeInventoryItem  InventoryConsumer
	CommitStudentStarOnce StarRewardCommitter
}
