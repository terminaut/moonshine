package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

func checkNotInFight(c echo.Context, userRepo *repository.UserRepository, userID uuid.UUID) error {
	inFight, err := userRepo.InFight(userID)
	if err != nil {
		return ErrInternalServerError(c)
	}
	if inFight {
		return ErrBadRequest(c, "user is in fight")
	}
	return nil
}

func ErrUnauthorized(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}

func ErrNotFound(c echo.Context, message string) error {
	if message == "" {
		message = "not found"
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": message})
}

func ErrBadRequest(c echo.Context, message string) error {
	if message == "" {
		message = "invalid request"
	}
	return c.JSON(http.StatusBadRequest, map[string]string{"error": message})
}

func ErrInternalServerError(c echo.Context) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func ErrConflict(c echo.Context, message string) error {
	if message == "" {
		message = "conflict"
	}
	return c.JSON(http.StatusConflict, map[string]string{"error": message})
}

func ErrUnauthorizedWithMessage(c echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": message})
}

func SuccessResponse(c echo.Context, message string) error {
	if message == "" {
		message = "ok"
	}
	return c.JSON(http.StatusOK, map[string]string{"message": message})
}

func resolveUserLocation(user *domain.User, locationRepo *repository.LocationRepository) *domain.Location {
	if user == nil || user.LocationID == uuid.Nil {
		return nil
	}
	location, _ := locationRepo.FindByID(user.LocationID)
	return location
}
