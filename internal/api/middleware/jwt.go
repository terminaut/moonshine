package middleware

import (
	"context"
	"reflect"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ExtractUserIDFromJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenValue := c.Get("user")
			if tokenValue == nil {
				return next(c)
			}

			var userID uuid.UUID
			var extracted bool

			rv := reflect.ValueOf(tokenValue)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}

			claimsField := rv.FieldByName("Claims")
			if claimsField.IsValid() && claimsField.CanInterface() {
				claims := claimsField.Interface()

				claimsValue := reflect.ValueOf(claims)
				if claimsValue.Kind() == reflect.Map {
					idValue := claimsValue.MapIndex(reflect.ValueOf("id"))
					if idValue.IsValid() && idValue.CanInterface() {
						idInterface := idValue.Interface()
						if idStr, ok := idInterface.(string); ok {
							if parsedID, err := uuid.Parse(idStr); err == nil {
								userID = parsedID
								extracted = true
							}
						}
					}
				}
			} else {
				if tokenInterface, ok := tokenValue.(interface{ Claims() jwt.Claims }); ok {
					claims := tokenInterface.Claims()

					if mapClaims, ok := claims.(jwt.MapClaims); ok {
						if idStr, ok := mapClaims["id"].(string); ok {
							if parsedID, err := uuid.Parse(idStr); err == nil {
								userID = parsedID
								extracted = true
							}
						}
					}
				} else {
					if t, ok := tokenValue.(*jwt.Token); ok {
						if claims, ok := t.Claims.(jwt.MapClaims); ok {
							if idStr, ok := claims["id"].(string); ok {
								if parsedID, err := uuid.Parse(idStr); err == nil {
									userID = parsedID
									extracted = true
								}
							}
						}
					}
				}
			}

			if extracted {
				ctx := context.WithValue(c.Request().Context(), userIDKey, userID)
				c.SetRequest(c.Request().WithContext(ctx))
			}

			return next(c)
		}
	}
}
