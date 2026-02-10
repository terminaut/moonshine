package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/domain"
	"moonshine/internal/repository"
	"moonshine/internal/worker"
)

type LocationHandler struct {
	db              *sqlx.DB
	locationService *services.LocationService
	locationRepo    *repository.LocationRepository
	userRepo        *repository.UserRepository
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

func NewLocationHandler(db *sqlx.DB, rdb *redis.Client) *LocationHandler {
	locationRepo := repository.NewLocationRepository(db)
	userRepo := repository.NewUserRepository(db)
	movingWorker := worker.NewCellsMovingWorker(locationRepo, userRepo, rdb, 5*time.Second)
	locationService, err := services.NewLocationService(db, rdb, locationRepo, userRepo, movingWorker)
	if err != nil {
		log.Fatalf("Failed to create LocationService: %v", err)
	}

	return &LocationHandler{
		db:              db,
		locationService: locationService,
		locationRepo:    locationRepo,
		userRepo:        userRepo,
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

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	err = h.locationService.MoveToLocation(c.Request().Context(), userID, locationSlug)
	if err != nil {
		switch err {
		case services.ErrLocationNotConnected:
			return ErrBadRequest(c, "locations not connected")
		case repository.ErrLocationNotFound:
			return ErrNotFound(c, "location not found")
		case repository.ErrUserNotFound:
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

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
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
		switch err {
		case services.ErrLocationNotConnected:
			return ErrBadRequest(c, "locations not connected")
		case repository.ErrLocationNotFound:
			return ErrNotFound(c, "location not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	if err := h.locationService.StartCellMovement(userID, path); err != nil {
		return ErrBadRequest(c, "")
	}

	targetLocation, err := h.locationRepo.FindBySlug(cellSlug)
	targetName := cellSlug
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

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	locationRepo := repository.NewLocationRepository(h.db)
	location, err := locationRepo.FindBySlug(locationSlug)
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
