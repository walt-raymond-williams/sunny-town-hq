package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInsufficientStars = errors.New("not enough stars")

type ShopPurchaseRequest struct {
	ShopID   string `json:"shopId"`
	ItemKey  string `json:"itemKey"`
	Quantity int    `json:"quantity"`
}

type ShopPurchaseResponse struct {
	StarBalance int             `json:"starBalance"`
	Inventory   StudentResponse `json:"inventory"`
}

func EnsureStudentWallet(ctx context.Context, db *pgxpool.Pool, userID int64) (int, error) {
	var balance int
	err := db.QueryRow(
		ctx,
		`
			insert into student_wallet (app_user_id)
			values ($1)
			on conflict (app_user_id) do update
			set updated_at = student_wallet.updated_at
			returning star_balance
		`,
		userID,
	).Scan(&balance)
	return balance, err
}

func PurchaseStudentShopItem(ctx context.Context, db *pgxpool.Pool, userID int64, request ShopPurchaseRequest) (ShopPurchaseResponse, error) {
	request.ShopID = strings.TrimSpace(request.ShopID)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if userID < 1 || request.Quantity < 1 {
		return ShopPurchaseResponse{}, errors.New("shop purchase is missing required fields")
	}
	if request.ShopID != "cookie-keeper-shop" || request.ItemKey != CookieKey {
		return ShopPurchaseResponse{}, errors.New("unsupported shop purchase")
	}

	totalPrice := 50 * request.Quantity
	tx, err := db.Begin(ctx)
	if err != nil {
		return ShopPurchaseResponse{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`insert into student_wallet (app_user_id) values ($1) on conflict (app_user_id) do nothing`,
		userID,
	); err != nil {
		return ShopPurchaseResponse{}, err
	}

	var starBalance int
	err = tx.QueryRow(
		ctx,
		`
			update student_wallet
			set star_balance = star_balance - $2,
				updated_at = now()
			where app_user_id = $1
				and star_balance >= $2
			returning star_balance
		`,
		userID,
		totalPrice,
	).Scan(&starBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShopPurchaseResponse{}, ErrInsufficientStars
	}
	if err != nil {
		return ShopPurchaseResponse{}, err
	}

	if _, err := tx.Exec(
		ctx,
		`
			insert into student_star_ledger (
				app_user_id,
				event_id,
				source,
				delta,
				metadata
			)
			values (
				$1,
				$2,
				'shop_purchase',
				$3,
				jsonb_build_object(
					'shop_id', $4::text,
					'item_key', $5::text,
					'quantity', $6::integer
				)
			)
		`,
		userID,
		fmt.Sprintf("shop-purchase:%d:%d", userID, time.Now().UnixNano()),
		-totalPrice,
		request.ShopID,
		request.ItemKey,
		request.Quantity,
	); err != nil {
		return ShopPurchaseResponse{}, err
	}

	if err := IncrementStudentItem(ctx, tx, userID, request.ItemKey, request.Quantity); err != nil {
		return ShopPurchaseResponse{}, err
	}
	inventory, err := LoadStudent(ctx, tx, userID)
	if err != nil {
		return ShopPurchaseResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ShopPurchaseResponse{}, err
	}

	return ShopPurchaseResponse{
		StarBalance: starBalance,
		Inventory:   inventory,
	}, nil
}
