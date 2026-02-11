package handlers

import (
	"bytes"
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

	"context"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupUserHandlerTest(t *testing.T) (*UserHandler, *sqlx.DB, *domain.User, echo.Echo) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewUserHandler(db, nil)
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
		Attack:     1, Defense: 1, Hp: 20, CurrentHp: 20, Level: 1, Gold: 100, Exp: 0, FreeStats: 0,
	}
	userRepo := repository.NewUserRepository(db)
	require.NoError(t, userRepo.Create(user))
	e := echo.New()
	return handler, db, user, *e
}

func ctxWithUserID(userID uuid.UUID) context.Context {
	return middleware.ContextWithUserID(context.Background(), userID)
}

func TestUserHandler_GetCurrentUser(t *testing.T) {
	handler, _, user, e := setupUserHandlerTest(t)

	t.Run("unauthorized when no userID in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success returns 200 and user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
		req = req.WithContext(ctxWithUserID(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var u dto.User
		err = json.Unmarshal(rec.Body.Bytes(), &u)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), u.ID)
		assert.Equal(t, user.Username, u.Username)
	})

	t.Run("404 when user not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
		req = req.WithContext(ctxWithUserID(uuid.New()))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHandler_GetUserInventory(t *testing.T) {
	handler, _, user, e := setupUserHandlerTest(t)

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me/inventory", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetUserInventory(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success returns 200 and array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me/inventory", nil)
		req = req.WithContext(ctxWithUserID(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetUserInventory(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var arr []dto.EquipmentItem
		err = json.Unmarshal(rec.Body.Bytes(), &arr)
		require.NoError(t, err)
		assert.NotNil(t, arr)
	})
}

func TestUserHandler_GetUserEquippedItems(t *testing.T) {
	handler, _, user, e := setupUserHandlerTest(t)

	t.Run("unauthorized when no userID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me/equipped", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetUserEquippedItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success returns 200 and map", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me/equipped", nil)
		req = req.WithContext(ctxWithUserID(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetUserEquippedItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var m map[string]dto.EquipmentItem
		err = json.Unmarshal(rec.Body.Bytes(), &m)
		require.NoError(t, err)
		assert.NotNil(t, m)
	})

	t.Run("404 when user not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me/equipped", nil)
		req = req.WithContext(ctxWithUserID(uuid.New()))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetUserEquippedItems(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHandler_UpdateCurrentUser(t *testing.T) {
	handler, db, user, e := setupUserHandlerTest(t)

	avatarRepo := repository.NewAvatarRepository(db)
	avatars, err := avatarRepo.FindAll()
	require.NoError(t, err)
	var avatarID string
	var avatarImage string
	if len(avatars) > 0 {
		avatarID = avatars[0].ID.String()
		avatarImage = avatars[0].Image
	} else {
		a := &domain.Avatar{Image: "img", Private: false}
		require.NoError(t, avatarRepo.Create(a))
		avatarID = a.ID.String()
		avatarImage = a.Image
	}

	t.Run("unauthorized when no userID", func(t *testing.T) {
		body := map[string]string{"avatarId": avatarID}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/api/user/me", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid avatar ID returns 400", func(t *testing.T) {
		body := map[string]string{"avatarId": "not-a-uuid"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/api/user/me", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctxWithUserID(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("success with avatarId returns 200", func(t *testing.T) {
		body := map[string]string{"avatarId": avatarID}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/api/user/me", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctxWithUserID(user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UpdateCurrentUser(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var got dto.User
		err = json.Unmarshal(rec.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, avatarImage, got.Avatar)
	})
}
