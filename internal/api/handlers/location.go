package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type LocationHandler struct {
	locationService *services.LocationService
	locationRepo    *repository.LocationRepository
	userRepo        *repository.UserRepository
	fightChecker    *FightChecker
}

type LocationCellsResponse struct {
	Cells []domain.LocationCell `json:"cells"`
}

type MoveToCellResponse struct {
	Message     string `json:"message"`
	PathLength  int    `json:"path_length"`
	TargetCell  string `json:"target_cell"`
	TimePerCell int    `json:"time_per_cell"`
}

func NewLocationHandler(
	locationService *services.LocationService,
	locationRepo *repository.LocationRepository,
	userRepo *repository.UserRepository,
	fightChecker *FightChecker,
) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
		locationRepo:    locationRepo,
		userRepo:        userRepo,
		fightChecker:    fightChecker,
	}
}

func (h *LocationHandler) MoveToLocation(c echo.Context) error {
	locationSlug := c.Param("slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.locationService.MoveToLocation(c.Request().Context(), userID, locationSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLocationNotConnected):
			return ErrBadRequest(c, "locations not connected")
		case errors.Is(err, repository.ErrLocationNotFound):
			return ErrNotFound(c, "location not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func (h *LocationHandler) MoveToCell(c echo.Context) error {
	cellSlug := c.Param("cell_slug")
	if cellSlug == "" {
		return ErrBadRequest(c, "cell slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return ErrNotFound(c, "user not found")
	}

	currentLocation, err := h.locationRepo.FindByID(user.LocationID)
	if err != nil {
		return ErrNotFound(c, "location not found")
	}

	if currentLocation.Slug == cellSlug {
		return c.JSON(http.StatusOK, nil)
	}

	path, err := h.locationService.FindShortestPath(currentLocation.Slug, cellSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLocationNotConnected):
			return ErrBadRequest(c, "locations not connected")
		case errors.Is(err, repository.ErrLocationNotFound):
			return ErrNotFound(c, "location not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	if err := h.locationService.StartCellMovement(userID, path); err != nil {
		return ErrBadRequest(c, "")
	}

	targetLocation, err := h.locationRepo.FindBySlug(cellSlug)
	var targetName string
	if err == nil && targetLocation != nil {
		targetName = targetLocation.Name
	} else {
		targetName = strings.TrimSuffix(cellSlug, "cell")
	}

	return c.JSON(http.StatusOK, &MoveToCellResponse{
		Message:     "movement started",
		PathLength:  len(path),
		TargetCell:  targetName,
		TimePerCell: 5,
	})
}

func (h *LocationHandler) GetLocationCells(c echo.Context) error {
	locationSlug := c.Param("slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	location, err := h.locationRepo.FindBySlug(locationSlug)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			return ErrNotFound(c, "location not found")
		}
		return ErrInternalServerError(c)
	}

	cells, err := h.locationService.FetchCells(c.Request().Context(), location.ID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, &LocationCellsResponse{
		Cells: cells,
	})
}
