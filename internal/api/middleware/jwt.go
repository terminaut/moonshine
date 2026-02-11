package middleware

import (
	"context"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ExtractUserIDFromJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := c.Get("user").(*jwtv5.Token)
			if !ok || token == nil {
				return next(c)
			}

			claims, ok := token.Claims.(jwtv5.MapClaims)
			if !ok {
				return next(c)
			}

			idStr, ok := claims["id"].(string)
			if !ok {
				return next(c)
			}

			userID, err := uuid.Parse(idStr)
			if err != nil {
				return next(c)
			}

			ctx := context.WithValue(c.Request().Context(), userIDKey, userID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
