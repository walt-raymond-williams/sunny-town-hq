package main

import (
	"context"
	"net/http"

	hqpet "hq/internal/hq/pet"
	hqsunnytownbridge "hq/internal/hq/sunnytownbridge"

	"github.com/jackc/pgx/v5"
)

type sunnyTownRewardEventRequest = hqsunnytownbridge.RewardEventRequest
type sunnyTownRewardEventResponse = hqsunnytownbridge.RewardEventResponse
type sunnyTownResourceEventRequest = hqsunnytownbridge.ResourceEventRequest
type sunnyTownResourceEventResponse = hqsunnytownbridge.ResourceEventResponse
type sunnyTownPositionRequest = hqsunnytownbridge.PositionRequest
type sunnyTownPositionResponse = hqsunnytownbridge.PositionResponse
type sunnyTownMapObjectResponse = hqsunnytownbridge.MapObjectResponse
type sunnyTownMapObjectsResponse = hqsunnytownbridge.MapObjectsResponse
type sunnyTownPlaceMapObjectRequest = hqsunnytownbridge.PlaceMapObjectRequest
type sunnyTownRemoveMapObjectRequest = hqsunnytownbridge.RemoveMapObjectRequest
type starRewardRequest = hqsunnytownbridge.StarRewardRequest
type inventoryLedgerRequest = hqsunnytownbridge.InventoryLedgerRequest
type starRewardQuerier = interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const stoneBlockItemKey = hqsunnytownbridge.StoneBlockItemKey

var (
	errInvalidMapObject     = hqsunnytownbridge.ErrInvalidMapObject
	errMapObjectOccupied    = hqsunnytownbridge.ErrMapObjectOccupied
	errMapObjectNotFound    = hqsunnytownbridge.ErrMapObjectNotFound
	errMapObjectNotOwned    = hqsunnytownbridge.ErrMapObjectNotOwned
	errUnsupportedMapObject = hqsunnytownbridge.ErrUnsupportedMapObject
)

func (app *app) sunnyTownBridgeStore() hqsunnytownbridge.Store {
	return hqsunnytownbridge.Store{DB: app.db}
}

func (app *app) sunnyTownBridgeHandler() hqsunnytownbridge.HTTPHandler {
	return hqsunnytownbridge.NewHTTPHandler(app.sunnyTownBridgeStore(), app.sunnyTownServiceSecret)
}

func (app *app) handleSunnyTownRewardEvent(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleRewardEvent(w, r)
}

func (app *app) handleSunnyTownResourceEvent(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleResourceEvent(w, r)
}

func (app *app) handleInternalSunnyTownStudentEquipment(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleStudentEquipment(w, r)
}

func (app *app) handleInternalSunnyTownInventoryQuantity(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleInventoryQuantity(w, r)
}

func (app *app) handleInternalSunnyTownPlayerPosition(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandlePlayerPosition(w, r)
}

func (app *app) handleInternalSunnyTownMapObjects(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleMapObjects(w, r)
}

func (app *app) handleInternalSunnyTownPlaceMapObject(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandlePlaceMapObject(w, r)
}

func (app *app) handleInternalSunnyTownRemoveMapObject(w http.ResponseWriter, r *http.Request) {
	app.sunnyTownBridgeHandler().HandleRemoveMapObject(w, r)
}

func (app *app) commitSunnyTownReward(ctx context.Context, request sunnyTownRewardEventRequest) (sunnyTownRewardEventResponse, error) {
	return app.sunnyTownBridgeStore().CommitReward(ctx, request)
}

func (app *app) commitSunnyTownResource(ctx context.Context, request sunnyTownResourceEventRequest) (sunnyTownResourceEventResponse, error) {
	return app.sunnyTownBridgeStore().CommitResource(ctx, request)
}

func (app *app) loadSunnyTownPosition(ctx context.Context, appUserID int64) (sunnyTownPositionResponse, error) {
	return app.sunnyTownBridgeStore().LoadPosition(ctx, appUserID)
}

func (app *app) saveSunnyTownPosition(ctx context.Context, request sunnyTownPositionRequest) (sunnyTownPositionResponse, error) {
	return app.sunnyTownBridgeStore().SavePosition(ctx, request)
}

func (app *app) loadSunnyTownMapObjects(ctx context.Context, roomID string, mapID string) (sunnyTownMapObjectsResponse, error) {
	return app.sunnyTownBridgeStore().LoadMapObjects(ctx, roomID, mapID)
}

func (app *app) placeSunnyTownMapObject(ctx context.Context, request sunnyTownPlaceMapObjectRequest) (sunnyTownMapObjectResponse, error) {
	return app.sunnyTownBridgeStore().PlaceMapObject(ctx, request)
}

func (app *app) removeSunnyTownMapObject(ctx context.Context, request sunnyTownRemoveMapObjectRequest) (sunnyTownMapObjectResponse, error) {
	return app.sunnyTownBridgeStore().RemoveMapObject(ctx, request)
}

func loadStudentInventoryQuantity(ctx context.Context, querier starRewardQuerier, userID int64, itemKey string) (int, error) {
	return hqsunnytownbridge.LoadStudentInventoryQuantity(ctx, querier, userID, itemKey)
}

func commitStudentStarReward(ctx context.Context, querier starRewardQuerier, request starRewardRequest) (bool, int, error) {
	return hqsunnytownbridge.CommitStudentStarReward(ctx, querier, request)
}

func commitStudentInventoryLedgerDelta(ctx context.Context, querier starRewardQuerier, request inventoryLedgerRequest) (bool, int, error) {
	return hqsunnytownbridge.CommitStudentInventoryLedgerDelta(ctx, querier, request)
}

func commitPetStarReward(ctx context.Context, tx pgx.Tx, request hqpet.StarRewardRequest) (bool, int, error) {
	return commitStudentStarReward(ctx, tx, starRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        request.Source,
		Delta:         request.Delta,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
}

func isSunnyTownFacing(value string) bool {
	return hqsunnytownbridge.IsFacing(value)
}

func isUniqueViolation(err error) bool {
	return hqsunnytownbridge.IsUniqueViolation(err)
}

func statusForMapObjectError(err error) int {
	return hqsunnytownbridge.StatusForMapObjectError(err)
}

func mapObjectErrorMessage(err error) string {
	return hqsunnytownbridge.MapObjectErrorMessage(err)
}
