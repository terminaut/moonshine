package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupEquipmentItemHandlerTest(t *testing.T) (*EquipmentItemHandler, *sqlx.DB, *domain.User, *domain.EquipmentItem, echo.Echo) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewEquipmentItemHandler(db)
	loc := &domain.Location{
		Name:     fmt.Sprintf("Loc %d", time.Now().UnixNano()),
		Slug:     fmt.Sprintf("loc-%d", time.Now().UnixNano()),
		Cell:     false,
		Inactive: false,
	}
	locRepo := repository.NewLocationRepository(db)
	require.NoError(t, locRepo.Create(loc))
	user := &domain.User{
		Username:   fmt.Sprintf("u%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("u%d@x.com", time.Now().UnixNano()),
		Password:   "x",
		Name:       "User",
		LocationID: loc.ID,
		Attack:     1, Defense: 1, Hp: 20, CurrentHp: 20, Level: 5, Gold: 500, Exp: 0, FreeStats: 0,
	}
	userRepo := repository.NewUserRepository(db)
	require.NoError(t, userRepo.Create(user))

	var categoryID uuid.UUID
	err := db.QueryRow(`INSERT INTO equipment_categories (name, type) VALUES ($1, $2::equipment_category_type) RETURNING id`, "Weapon", "weapon").Scan(&categoryID)
	require.NoError(t, err)

	item := &domain.EquipmentItem{
		Name:   "Test Sword",
		Slug:   fmt.Sprintf("sword-%d", time.Now().UnixNano()),
		Attack: 5, Defense: 2, Hp: 10, RequiredLevel: 1, Price: 100,
		EquipmentCategoryID: categoryID,
	}
	itemRepo := repository.NewEquipmentItemRepository(db)
	require.NoError(t, itemRepo.Create(item))

	inventoryRepo := repository.NewInventoryRepository(db)
	require.NoError(t, inventoryRepo.Create(&domain.Inventory{UserID: user.ID, EquipmentItemID: item.ID}))

	e := echo.New()
	return handler, db, user, item, *e
}

func eqCtx(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID)
}

func TestEquipmentItemHandler_GetEquipmentItems(t *testing.T) {
	handler, _, user, _, e := setupEquipmentItemHandlerTest(t)

	t.Run("missing category returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/equipment_items", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetEquipmentItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/equipment_items?category=weapon", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetEquipmentItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success returns 200 and items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/equipment_items?category=weapon", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetEquipmentItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var items []dto.EquipmentItem
		err = json.Unmarshal(rec.Body.Bytes(), &items)
		require.NoError(t, err)
		assert.NotNil(t, items)
	})
}

func TestEquipmentItemHandler_BuyEquipmentItem(t *testing.T) {
	handler, db, user, item, e := setupEquipmentItemHandlerTest(t)

	t.Run("empty slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items//buy", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/buy")
		c.SetParamNames("slug")
		c.SetParamValues("")

		err := handler.BuyEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/x/buy", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/buy")
		c.SetParamNames("slug")
		c.SetParamValues("x")

		err := handler.BuyEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success buy returns 200", func(t *testing.T) {
		newItem := &domain.EquipmentItem{
			Name:   "Cheap",
			Slug:   fmt.Sprintf("cheap-%d", time.Now().UnixNano()),
			Attack: 0, Defense: 0, Hp: 0, RequiredLevel: 1, Price: 10,
			EquipmentCategoryID: item.EquipmentCategoryID,
		}
		itemRepo := repository.NewEquipmentItemRepository(db)
		require.NoError(t, itemRepo.Create(newItem))

		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/"+newItem.Slug+"/buy", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/buy")
		c.SetParamNames("slug")
		c.SetParamValues(newItem.Slug)

		err := handler.BuyEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestEquipmentItemHandler_TakeOnEquipmentItem(t *testing.T) {
	handler, _, user, item, e := setupEquipmentItemHandlerTest(t)

	t.Run("empty slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items//take_on", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/take_on")
		c.SetParamNames("slug")
		c.SetParamValues("")

		err := handler.TakeOnEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/x/take_on", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/take_on")
		c.SetParamNames("slug")
		c.SetParamValues("x")

		err := handler.TakeOnEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success take_on returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/"+item.Slug+"/take_on", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/take_on")
		c.SetParamNames("slug")
		c.SetParamValues(item.Slug)

		err := handler.TakeOnEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestEquipmentItemHandler_TakeOffEquipmentItem(t *testing.T) {
	handler, db, user, item, e := setupEquipmentItemHandlerTest(t)

	itemRepo := repository.NewEquipmentItemRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	invRepo.Create(&domain.Inventory{UserID: user.ID, EquipmentItemID: item.ID})
	takeOnSvc := NewEquipmentItemTakeOnService(db, itemRepo, invRepo, repository.NewUserRepository(db))
	err := takeOnSvc.TakeOnEquipmentItem(context.Background(), user.ID, item.ID)
	require.NoError(t, err)

	t.Run("empty slot returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/take_off/", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/take_off/:slot")
		c.SetParamNames("slot")
		c.SetParamValues("")

		err := handler.TakeOffEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/take_off/weapon", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/take_off/:slot")
		c.SetParamNames("slot")
		c.SetParamValues("weapon")

		err := handler.TakeOffEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success take_off returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/take_off/weapon", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/take_off/:slot")
		c.SetParamNames("slot")
		c.SetParamValues("weapon")

		err := handler.TakeOffEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestEquipmentItemHandler_SellEquipmentItem(t *testing.T) {
	handler, _, user, item, e := setupEquipmentItemHandlerTest(t)

	t.Run("empty slug returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items//sell", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/sell")
		c.SetParamNames("slug")
		c.SetParamValues("")

		err := handler.SellEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/x/sell", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/sell")
		c.SetParamNames("slug")
		c.SetParamValues("x")

		err := handler.SellEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success sell returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/equipment_items/"+item.Slug+"/sell", nil)
		req = req.WithContext(eqCtx(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/equipment_items/:slug/sell")
		c.SetParamNames("slug")
		c.SetParamValues(item.Slug)

		err := handler.SellEquipmentItem(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
