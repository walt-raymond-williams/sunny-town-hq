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
var ErrInsufficientShopStock = errors.New("not enough shop stock")

const CookieKeeperShopID = "cookie-keeper-shop"
const CookieKeeperCookieStockCapacity = 64
const CookieKeeperInputStorageCapacity = 64

type ShopStockEventRequest struct {
	EventID string
	Source  string
	ShopID  string
	ItemKey string
	Delta   int
}

type ShopRecipeProductionRequest struct {
	EventID   string
	Source    string
	ShopID    string
	RecipeKey string
	Amount    int
}

type ShopPurchaseRequest struct {
	ShopID   string `json:"shopId"`
	ItemKey  string `json:"itemKey"`
	Quantity int    `json:"quantity"`
}

type ShopPurchaseResponse struct {
	StarBalance int             `json:"starBalance"`
	Inventory   StudentResponse `json:"inventory"`
}

type ShopStockItemResponse struct {
	ItemKey  string `json:"itemKey"`
	Quantity int    `json:"quantity"`
	Capacity int    `json:"capacity"`
}

type ShopStockResponse struct {
	ShopID string                  `json:"shopId"`
	Items  []ShopStockItemResponse `json:"items"`
}

type ShopInputStorageItemResponse struct {
	ItemKey  string `json:"itemKey"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type ShopInputStorageResponse struct {
	ShopID   string                         `json:"shopId"`
	Capacity int                            `json:"capacity"`
	Items    []ShopInputStorageItemResponse `json:"items"`
}

type shopRecipeQuerier interface {
	Querier
	rowQuerier
}

type shopRecipeStorage struct {
	querier      shopRecipeQuerier
	shopID       string
	stockEventID string
	source       string
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
	if request.ShopID != CookieKeeperShopID || request.ItemKey != CookieKey {
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

	stockConsumed, err := ConsumeShopStockItem(ctx, tx, request.ShopID, request.ItemKey, request.Quantity)
	if err != nil {
		return ShopPurchaseResponse{}, err
	}
	if !stockConsumed {
		return ShopPurchaseResponse{}, ErrInsufficientShopStock
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

func LoadShopStock(ctx context.Context, querier Loader, shopID string) (ShopStockResponse, error) {
	shopID = strings.TrimSpace(shopID)
	if shopID == "" {
		return ShopStockResponse{}, errors.New("shop_id is required")
	}
	if shopID != CookieKeeperShopID {
		return ShopStockResponse{}, errors.New("unsupported shop")
	}

	rows, err := querier.Query(
		ctx,
		`
			select iit.key, coalesce(ssi.quantity, 0) as quantity
			from inventory_item_type iit
			left join shop_stock_item ssi on ssi.item_type_id = iit.id
				and ssi.shop_id = $1
			where iit.key = $2
			order by iit.id
		`,
		shopID,
		CookieKey,
	)
	if err != nil {
		return ShopStockResponse{}, err
	}
	defer rows.Close()

	response := ShopStockResponse{ShopID: shopID, Items: []ShopStockItemResponse{}}
	for rows.Next() {
		var item ShopStockItemResponse
		if err := rows.Scan(&item.ItemKey, &item.Quantity); err != nil {
			return ShopStockResponse{}, err
		}
		item.Capacity = stockCapacityForItem(shopID, item.ItemKey)
		response.Items = append(response.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ShopStockResponse{}, err
	}
	return response, nil
}

func CommitShopStockDelta(ctx context.Context, querier rowQuerier, request ShopStockEventRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ShopID = strings.TrimSpace(request.ShopID)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.EventID == "" || request.Source == "" || request.ShopID == "" || request.ItemKey == "" || request.Delta < 1 {
		return false, 0, errors.New("shop stock event is missing required fields")
	}
	if request.ShopID != CookieKeeperShopID || request.ItemKey != CookieKey {
		return false, 0, errors.New("unsupported shop stock item")
	}
	capacity := stockCapacityForItem(request.ShopID, request.ItemKey)

	var inserted bool
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			with item_type as (
				select id
				from inventory_item_type
				where key = $4
			),
			inserted as (
				insert into shop_stock_ledger (
					event_id,
					source,
					shop_id,
					item_type_id,
					delta
				)
				select $1, $2, $3, id, $5
				from item_type
				on conflict (event_id) do nothing
				returning shop_id, item_type_id, delta
			),
			updated_stock as (
				insert into shop_stock_item (shop_id, item_type_id, quantity)
				select shop_id, item_type_id, least($6::integer, delta) from inserted
				on conflict (shop_id, item_type_id) do update
				set quantity = least($6::integer, shop_stock_item.quantity + excluded.quantity),
					updated_at = now()
				returning quantity
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select quantity from updated_stock),
					(
						select ssi.quantity
						from shop_stock_item ssi
						join item_type on item_type.id = ssi.item_type_id
						where ssi.shop_id = $3
					),
					0
				) as quantity
		`,
		request.EventID,
		request.Source,
		request.ShopID,
		request.ItemKey,
		request.Delta,
		capacity,
	).Scan(&inserted, &quantity)
	return inserted, quantity, err
}

func stockCapacityForItem(shopID string, itemKey string) int {
	if shopID == CookieKeeperShopID && itemKey == CookieKey {
		return CookieKeeperCookieStockCapacity
	}
	return 0
}

func ConsumeShopStockItem(ctx context.Context, querier Querier, shopID string, itemKey string, quantity int) (bool, error) {
	shopID = strings.TrimSpace(shopID)
	itemKey = strings.TrimSpace(itemKey)
	if shopID == "" || itemKey == "" || quantity < 1 {
		return false, errors.New("shop stock consume is missing required fields")
	}

	result, err := querier.Exec(
		ctx,
		`
			update shop_stock_item ssi
			set quantity = quantity - $3,
				updated_at = now()
			from inventory_item_type iit
			where ssi.item_type_id = iit.id
				and ssi.shop_id = $1
				and iit.key = $2
				and ssi.quantity >= $3
		`,
		shopID,
		itemKey,
		quantity,
	)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func LoadShopInputStorage(ctx context.Context, querier Loader, shopID string) (ShopInputStorageResponse, error) {
	shopID = strings.TrimSpace(shopID)
	if shopID == "" {
		return ShopInputStorageResponse{}, errors.New("shop input storage shop_id is required")
	}
	if shopID != CookieKeeperShopID {
		return ShopInputStorageResponse{}, errors.New("unsupported shop input storage")
	}

	rows, err := querier.Query(
		ctx,
		`
			select iit.key,
				iit.name,
				sisi.quantity
			from shop_input_storage_item sisi
			join inventory_item_type iit on iit.id = sisi.item_type_id
			where sisi.shop_id = $1
				and sisi.quantity > 0
			order by iit.id
		`,
		shopID,
	)
	if err != nil {
		return ShopInputStorageResponse{}, err
	}
	defer rows.Close()

	response := ShopInputStorageResponse{
		ShopID:   shopID,
		Capacity: shopInputStorageCapacity(shopID),
		Items:    []ShopInputStorageItemResponse{},
	}
	for rows.Next() {
		var item ShopInputStorageItemResponse
		if err := rows.Scan(&item.ItemKey, &item.Name, &item.Quantity); err != nil {
			return ShopInputStorageResponse{}, err
		}
		response.Items = append(response.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ShopInputStorageResponse{}, err
	}
	return response, nil
}

func IncrementShopInputStorageItem(ctx context.Context, querier rowQuerier, shopID string, itemKey string, quantity int) (bool, int, error) {
	shopID = strings.TrimSpace(shopID)
	itemKey = strings.TrimSpace(itemKey)
	if shopID == "" || itemKey == "" || quantity < 1 {
		return false, 0, errors.New("shop input storage increment is missing required fields")
	}
	if shopID != CookieKeeperShopID {
		return false, 0, errors.New("unsupported shop input storage")
	}
	capacity := shopInputStorageCapacity(shopID)

	var accepted bool
	var totalQuantity int
	err := querier.QueryRow(
		ctx,
		`
			with item_type as (
				select id
				from inventory_item_type
				where key = $2
			),
			shop_lock as (
				select pg_advisory_xact_lock(hashtext('shop-input-storage:' || $1))
			),
			current_total as (
				select coalesce(sum(quantity), 0)::integer as quantity
				from shop_input_storage_item, shop_lock
				where shop_id = $1
			),
			upserted as (
				insert into shop_input_storage_item (shop_id, item_type_id, quantity)
				select $1, item_type.id, $3
				from item_type, current_total
				where current_total.quantity + $3 <= $4
				on conflict (shop_id, item_type_id) do update
				set quantity = shop_input_storage_item.quantity + excluded.quantity,
					updated_at = now()
				returning quantity
			)
			select exists(select 1 from upserted) as accepted,
				(
					select quantity from current_total
				) + case when exists(select 1 from upserted) then $3 else 0 end as total_quantity
		`,
		shopID,
		itemKey,
		quantity,
		capacity,
	).Scan(&accepted, &totalQuantity)
	return accepted, totalQuantity, err
}

func ConsumeShopInputStorageItem(ctx context.Context, querier Querier, shopID string, itemKey string, quantity int) (bool, error) {
	shopID = strings.TrimSpace(shopID)
	itemKey = strings.TrimSpace(itemKey)
	if shopID == "" || itemKey == "" || quantity < 1 {
		return false, errors.New("shop input storage consume is missing required fields")
	}
	if shopID != CookieKeeperShopID {
		return false, errors.New("unsupported shop input storage")
	}

	result, err := querier.Exec(
		ctx,
		`
			update shop_input_storage_item sisi
			set quantity = quantity - $3,
				updated_at = now()
			from inventory_item_type iit
			where sisi.item_type_id = iit.id
				and sisi.shop_id = $1
				and iit.key = $2
				and sisi.quantity >= $3
		`,
		shopID,
		itemKey,
		quantity,
	)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func shopInputStorageCapacity(shopID string) int {
	if shopID == CookieKeeperShopID {
		return CookieKeeperInputStorageCapacity
	}
	return 0
}

func CommitShopRecipeProduction(ctx context.Context, querier shopRecipeQuerier, request ShopRecipeProductionRequest) error {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ShopID = strings.TrimSpace(request.ShopID)
	request.RecipeKey = strings.TrimSpace(request.RecipeKey)
	if request.EventID == "" || request.Source == "" || request.ShopID == "" || request.RecipeKey == "" || request.Amount < 1 {
		return errors.New("shop recipe production is missing required fields")
	}
	if request.ShopID != CookieKeeperShopID || request.RecipeKey != CookieRecipeKey {
		return errors.New("unsupported shop recipe production")
	}

	recipe, ok := recipeByKey(request.RecipeKey)
	if !ok {
		return ErrUnknownRecipe
	}

	storage := shopRecipeStorage{
		querier:      querier,
		shopID:       request.ShopID,
		stockEventID: request.EventID,
		source:       request.Source,
	}
	return executeRecipe(ctx, scaledRecipe(recipe, request.Amount), storage)
}

func (storage shopRecipeStorage) PrepareRecipe(ctx context.Context, recipe RecipeDefinition) error {
	var locked int
	if err := storage.querier.QueryRow(
		ctx,
		`select 1 from (select pg_advisory_xact_lock(hashtext('shop-recipe-production:' || $1))) as locked`,
		storage.shopID,
	).Scan(&locked); err != nil {
		return err
	}

	capacity := stockCapacityForItem(storage.shopID, recipe.OutputKey)
	if capacity < 1 {
		return errors.New("unsupported shop recipe output")
	}

	var currentQuantity int
	if err := storage.querier.QueryRow(
		ctx,
		`
			select coalesce(ssi.quantity, 0)
			from inventory_item_type iit
			left join shop_stock_item ssi on ssi.item_type_id = iit.id
				and ssi.shop_id = $1
			where iit.key = $2
		`,
		storage.shopID,
		recipe.OutputKey,
	).Scan(&currentQuantity); err != nil {
		return err
	}
	if currentQuantity+recipe.Quantity > capacity {
		return ErrRecipeOutputFull
	}

	for _, ingredient := range recipe.Ingredients {
		var inputQuantity int
		if err := storage.querier.QueryRow(
			ctx,
			`
				select coalesce(sisi.quantity, 0)
				from inventory_item_type iit
				left join shop_input_storage_item sisi on sisi.item_type_id = iit.id
					and sisi.shop_id = $1
				where iit.key = $2
			`,
			storage.shopID,
			ingredient.ItemKey,
		).Scan(&inputQuantity); err != nil {
			return err
		}
		if inputQuantity < ingredient.Quantity {
			return ErrInsufficientIngredient
		}
	}
	return nil
}

func (storage shopRecipeStorage) ConsumeRecipeItem(ctx context.Context, itemKey string, quantity int) (bool, error) {
	return ConsumeShopInputStorageItem(ctx, storage.querier, storage.shopID, itemKey, quantity)
}

func (storage shopRecipeStorage) ProduceRecipeItem(ctx context.Context, itemKey string, quantity int) error {
	_, _, err := CommitShopStockDelta(ctx, storage.querier, ShopStockEventRequest{
		EventID: storage.stockEventID,
		Source:  storage.source,
		ShopID:  storage.shopID,
		ItemKey: itemKey,
		Delta:   quantity,
	})
	return err
}
