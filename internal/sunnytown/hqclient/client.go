package hqclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	stprotocol "hq/internal/sunnytown/protocol"
)

type Client struct {
	baseURL       string
	serviceSecret string
	httpClient    *http.Client
}

func New(baseURL string, serviceSecret string, timeout time.Duration) *Client {
	return &Client{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		serviceSecret: strings.TrimSpace(serviceSecret),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type RewardCommitRequest struct {
	EventID       string `json:"event_id"`
	AppUserID     int64  `json:"app_user_id"`
	RoomID        string `json:"room_id"`
	MapID         string `json:"map_id"`
	CollectibleID string `json:"collectible_id"`
	RewardKind    string `json:"reward_kind"`
	Amount        int    `json:"amount"`
}

type RewardCommitResponse struct {
	Accepted       bool `json:"accepted"`
	Duplicate      bool `json:"duplicate"`
	NewStarBalance int  `json:"new_star_balance"`
}

type ResourceCommitRequest struct {
	EventID     string `json:"event_id"`
	AppUserID   int64  `json:"app_user_id"`
	Source      string `json:"source"`
	RoomID      string `json:"room_id"`
	MapID       string `json:"map_id"`
	NodeID      string `json:"node_id"`
	ResourceKey string `json:"resource_key"`
	Amount      int    `json:"amount"`
}

type ResourceCommitResponse struct {
	Accepted    bool   `json:"accepted"`
	Duplicate   bool   `json:"duplicate"`
	ResourceKey string `json:"resource_key"`
	Quantity    int    `json:"quantity"`
}

type MapObjectsResponse struct {
	Objects []MapObjectResponse `json:"objects"`
}

type MapObjectResponse struct {
	ID                  int64  `json:"id"`
	RoomID              string `json:"room_id"`
	MapID               string `json:"map_id"`
	GridX               int    `json:"grid_x"`
	GridY               int    `json:"grid_y"`
	ItemKey             string `json:"item_key"`
	PlacedByAppUserID   int64  `json:"placed_by_app_user_id"`
	RemainingItemAmount int    `json:"remaining_item_amount"`
}

type PlaceMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
	ItemKey   string `json:"item_key"`
}

type RemoveMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
}

type StudentPositionResponse struct {
	Found     bool    `json:"found"`
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type StudentPositionRequest struct {
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type studentEquipmentResponse struct {
	Slots []equipmentSlotResponse `json:"slots"`
}

type inventoryQuantityResponse struct {
	ItemKey  string `json:"item_key"`
	Quantity int    `json:"quantity"`
}

type equipmentSlotResponse struct {
	Slot string                 `json:"slot"`
	Item *equipmentItemResponse `json:"item"`
}

type equipmentItemResponse struct {
	VisualKey string `json:"visualKey"`
}

func (client *Client) LoadStudentEquipment(ctx context.Context, appUserID int64) (stprotocol.EquipmentSnapshot, error) {
	response, err := client.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/student-equipment?app_user_id="+strconv.FormatInt(appUserID, 10),
		nil,
	)
	if err != nil {
		return stprotocol.EquipmentSnapshot{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return stprotocol.EquipmentSnapshot{}, fmt.Errorf("equipment request failed status=%d", response.StatusCode)
	}

	var equipment studentEquipmentResponse
	if err := json.NewDecoder(response.Body).Decode(&equipment); err != nil {
		return stprotocol.EquipmentSnapshot{}, err
	}

	snapshot := stprotocol.EquipmentSnapshot{}
	for _, slot := range equipment.Slots {
		if slot.Slot != "gear" && slot.Slot != "accessory" && slot.Slot != "tool" {
			continue
		}
		if slot.Item == nil || strings.TrimSpace(slot.Item.VisualKey) == "" {
			continue
		}
		snapshot[slot.Slot] = strings.TrimSpace(slot.Item.VisualKey)
	}
	return snapshot, nil
}

func (client *Client) LoadStudentInventoryQuantity(ctx context.Context, appUserID int64, itemKey string) (int, error) {
	query := url.Values{}
	query.Set("app_user_id", strconv.FormatInt(appUserID, 10))
	query.Set("item_key", itemKey)
	response, err := client.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/inventory-quantity?"+query.Encode(),
		nil,
	)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return 0, fmt.Errorf("inventory quantity request failed status=%d", response.StatusCode)
	}

	var quantity inventoryQuantityResponse
	if err := json.NewDecoder(response.Body).Decode(&quantity); err != nil {
		return 0, err
	}
	return quantity.Quantity, nil
}

func (client *Client) LoadStudentPosition(ctx context.Context, appUserID int64) (StudentPositionResponse, error) {
	response, err := client.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/player-position?app_user_id="+strconv.FormatInt(appUserID, 10),
		nil,
	)
	if err != nil {
		return StudentPositionResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return StudentPositionResponse{}, fmt.Errorf("position request failed status=%d", response.StatusCode)
	}

	var position StudentPositionResponse
	if err := json.NewDecoder(response.Body).Decode(&position); err != nil {
		return StudentPositionResponse{}, err
	}
	return position, nil
}

func (client *Client) SaveStudentPosition(ctx context.Context, position StudentPositionRequest) error {
	response, err := client.jsonRequest(ctx, http.MethodPost, "/api/internal/sunny-town/player-position", position)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("save position failed status=%d", response.StatusCode)
	}
	return nil
}

func (client *Client) LoadMapObjects(ctx context.Context, roomID string, mapID string) (MapObjectsResponse, error) {
	query := url.Values{}
	query.Set("room_id", roomID)
	query.Set("map_id", mapID)
	response, err := client.internalRequest(ctx, http.MethodGet, "/api/internal/sunny-town/map-objects?"+query.Encode(), nil)
	if err != nil {
		return MapObjectsResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return MapObjectsResponse{}, fmt.Errorf("map objects request failed status=%d", response.StatusCode)
	}

	var loaded MapObjectsResponse
	if err := json.NewDecoder(response.Body).Decode(&loaded); err != nil {
		return MapObjectsResponse{}, err
	}
	return loaded, nil
}

func (client *Client) PlaceMapObject(ctx context.Context, request PlaceMapObjectRequest) (MapObjectResponse, error) {
	response, err := client.jsonRequest(ctx, http.MethodPost, "/api/internal/sunny-town/map-objects/place", request)
	if err != nil {
		return MapObjectResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return MapObjectResponse{}, fmt.Errorf("place map object failed status=%d", response.StatusCode)
	}

	var placed MapObjectResponse
	if err := json.NewDecoder(response.Body).Decode(&placed); err != nil {
		return MapObjectResponse{}, err
	}
	return placed, nil
}

func (client *Client) RemoveMapObject(ctx context.Context, request RemoveMapObjectRequest) (MapObjectResponse, error) {
	response, err := client.jsonRequest(ctx, http.MethodPost, "/api/internal/sunny-town/map-objects/remove", request)
	if err != nil {
		return MapObjectResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return MapObjectResponse{}, fmt.Errorf("remove map object failed status=%d", response.StatusCode)
	}

	var removed MapObjectResponse
	if err := json.NewDecoder(response.Body).Decode(&removed); err != nil {
		return MapObjectResponse{}, err
	}
	return removed, nil
}

func (client *Client) CommitReward(ctx context.Context, request RewardCommitRequest) (RewardCommitResponse, error) {
	response, err := client.jsonRequest(ctx, http.MethodPost, "/api/internal/sunny-town/reward-events", request)
	if err != nil {
		return RewardCommitResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return RewardCommitResponse{}, fmt.Errorf("hq reward status %d", response.StatusCode)
	}

	var committed RewardCommitResponse
	if err := json.NewDecoder(response.Body).Decode(&committed); err != nil {
		return RewardCommitResponse{}, err
	}
	if !committed.Accepted {
		return RewardCommitResponse{}, errors.New("hq rejected reward")
	}
	return committed, nil
}

func (client *Client) CommitResource(ctx context.Context, request ResourceCommitRequest) (ResourceCommitResponse, error) {
	response, err := client.jsonRequest(ctx, http.MethodPost, "/api/internal/sunny-town/resource-events", request)
	if err != nil {
		return ResourceCommitResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ResourceCommitResponse{}, fmt.Errorf("hq resource status %d", response.StatusCode)
	}

	var committed ResourceCommitResponse
	if err := json.NewDecoder(response.Body).Decode(&committed); err != nil {
		return ResourceCommitResponse{}, err
	}
	if !committed.Accepted {
		return ResourceCommitResponse{}, errors.New("hq rejected resource")
	}
	return committed, nil
}

func (client *Client) jsonRequest(ctx context.Context, method string, path string, body any) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return client.internalRequest(ctx, method, path, encoded)
}

func (client *Client) internalRequest(ctx context.Context, method string, path string, body []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		client.baseURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-HQ-Service-Secret", client.serviceSecret)
	return client.httpClient.Do(request)
}
