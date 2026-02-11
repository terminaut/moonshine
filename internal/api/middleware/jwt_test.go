package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestToken(claims jwtv5.MapClaims, signingKey string) *jwtv5.Token {
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	// Sign the token so it's marked as valid
	token.Raw, _ = token.SignedString([]byte(signingKey))
	token.Valid = true
	return token
}

func TestExtractUserIDFromJWT(t *testing.T) {
	middleware := ExtractUserIDFromJWT()

	t.Run("valid token sets user ID in context", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		userID := uuid.New()
		token := createTestToken(jwtv5.MapClaims{
			"id":  userID.String(),
			"exp": time.Now().Add(72 * time.Hour).Unix(),
		}, "test-secret")
		c.Set("user", token)

		var extractedID uuid.UUID
		handler := middleware(func(c echo.Context) error {
			var err error
			extractedID, err = GetUserIDFromContext(c.Request().Context())
			require.NoError(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, userID, extractedID)
	})

	t.Run("no token in context passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("wrong type in context passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", "not-a-token")

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("nil token passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", (*jwtv5.Token)(nil))

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("token without id claim passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		token := createTestToken(jwtv5.MapClaims{
			"exp": time.Now().Add(72 * time.Hour).Unix(),
		}, "test-secret")
		c.Set("user", token)

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("token with non-string id claim passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		token := createTestToken(jwtv5.MapClaims{
			"id":  12345,
			"exp": time.Now().Add(72 * time.Hour).Unix(),
		}, "test-secret")
		c.Set("user", token)

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("token with invalid UUID passes through", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		token := createTestToken(jwtv5.MapClaims{
			"id":  "not-a-uuid",
			"exp": time.Now().Add(72 * time.Hour).Unix(),
		}, "test-secret")
		c.Set("user", token)

		called := false
		handler := middleware(func(c echo.Context) error {
			called = true
			_, err := GetUserIDFromContext(c.Request().Context())
			assert.Error(t, err)
			return nil
		})

		err := handler(c)
		require.NoError(t, err)
		assert.True(t, called)
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("uuid.UUID value in context", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), userIDKey, userID)

		result, err := GetUserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, result)
	})

	t.Run("string UUID value in context", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), userIDKey, userID.String())

		result, err := GetUserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, result)
	})

	t.Run("invalid string value in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "not-a-uuid")

		_, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
	})

	t.Run("no value in context", func(t *testing.T) {
		_, err := GetUserIDFromContext(context.Background())
		assert.Error(t, err)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, 12345)

		_, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
	})
}
