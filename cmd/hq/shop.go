package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
)

var errInsufficientStars = hqinventory.ErrInsufficientStars

type shopPurchaseRequest = hqinventory.ShopPurchaseRequest
type shopPurchaseResponse = hqinventory.ShopPurchaseResponse

func (app *app) ensureStudentWallet(ctx context.Context, userID int64) (int, error) {
	return hqinventory.EnsureStudentWallet(ctx, app.db, userID)
}

func (app *app) purchaseStudentShopItem(ctx context.Context, userID int64, request shopPurchaseRequest) (shopPurchaseResponse, error) {
	return hqinventory.PurchaseStudentShopItem(ctx, app.db, userID, request)
}
