package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func setupAvatarHandlerTest(t *testing.T) (*AvatarHandler, *domain.User, echo.Echo) {
	if testDB == nil {
		t.Skip("Test database not initialized")
	}
	db := testDB.DB()
	handler := NewAvatarHandler(db)
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
		Attack:     1, Defense: 1, Hp: 20, CurrentHp: 20, Level: 1, Gold: 0, Exp: 0, FreeStats: 0,
	}
	userRepo := repository.NewUserRepository(db)
	require.NoError(t, userRepo.Create(user))
	e := echo.New()
	return handler, user, *e
}

func TestAvatarHandler_GetAllAvatars(t *testing.T) {
	handler, user, e := setupAvatarHandlerTest(t)

	t.Run("unauthorized when no userID in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/avatars", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetAllAvatars(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("success returns 200 and avatars array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/avatars", nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), user.ID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.GetAllAvatars(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var arr []dto.Avatar
		err = json.Unmarshal(rec.Body.Bytes(), &arr)
		require.NoError(t, err)
		assert.NotNil(t, arr)
	})
}
