package handlers

import (
	"errors"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type FightHandler struct {
	fightService *services.FightService
	locationRepo *repository.LocationRepository
}

func NewFightHandler(db *sqlx.DB) *FightHandler {
	locationRepo := repository.NewLocationRepository(db)
	fightService := services.NewFightService(
		db,
		repository.NewFightRepository(db),
		repository.NewBotRepository(db),
		repository.NewUserRepository(db),
		repository.NewRoundRepository(db),
	)

	return &FightHandler{
		fightService: fightService,
		locationRepo: locationRepo,
	}
}

func handleFightError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrNoActiveFight):
		return ErrNotFound(c, "no active fight")
	case errors.Is(err, services.ErrUserNotFound):
		return ErrNotFound(c, "user not found")
	case errors.Is(err, services.ErrBotNotFound):
		return ErrNotFound(c, "bot not found")
	case errors.Is(err, services.ErrInvalidBodyPart):
		return ErrBadRequest(c, "invalid body part")
	default:
		return ErrInternalServerError(c)
	}
}

type GetCurrentFightResponse struct {
	User  dto.User  `json:"user"`
	Bot   dto.Bot   `json:"bot"`
	Fight dto.Fight `json:"fight"`
}

func (h *FightHandler) fightResponse(c echo.Context, result *services.GetCurrentFightResult) error {
	if result == nil {
		return ErrInternalServerError(c)
	}

	location := resolveUserLocation(result.User, h.locationRepo)
	userDTO := dto.UserFromDomain(result.User, location, nil, true)
	botDTO := dto.BotFromDomain(result.Bot)
	fightDTO := dto.FightFromDomain(result.Fight)

	if userDTO == nil || botDTO == nil || fightDTO == nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, &GetCurrentFightResponse{
		User:  *userDTO,
		Bot:   *botDTO,
		Fight: *fightDTO,
	})
}

func (h *FightHandler) GetCurrentFight(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	result, err := h.fightService.GetCurrentFight(c.Request().Context(), userID)
	if err != nil {
		return handleFightError(c, err)
	}

	return h.fightResponse(c, result)
}

type HitRequest struct {
	Attack  string `json:"attack" validate:"required"`
	Defense string `json:"defense" validate:"required"`
}

func (h *FightHandler) Hit(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	var req HitRequest
	if err := c.Bind(&req); err != nil {
		return ErrBadRequest(c, "invalid request")
	}
	if err := c.Validate(&req); err != nil {
		return ErrBadRequest(c, err.Error())
	}

	result, err := h.fightService.Hit(c.Request().Context(), userID, req.Attack, req.Defense)
	if err != nil {
		return handleFightError(c, err)
	}

	return h.fightResponse(c, result)
}
