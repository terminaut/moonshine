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

type BotHandler struct {
	botService *services.BotService
	userRepo   *repository.UserRepository
}

type BotResponse struct {
	Bots []*dto.Bot `json:"bots"`
}

func NewBotHandler(db *sqlx.DB) *BotHandler {
	userRepo := repository.NewUserRepository(db)
	botService := services.NewBotService(
		repository.NewLocationRepository(db),
		repository.NewBotRepository(db),
		userRepo,
		repository.NewFightRepository(db),
		repository.NewRoundRepository(db),
	)

	return &BotHandler{
		botService: botService,
		userRepo:   userRepo,
	}
}

func (h *BotHandler) GetBots(c echo.Context) error {
	locationSlug := c.Param("location_slug")
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

	bots, err := h.botService.GetBotsByLocationSlug(locationSlug)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			return ErrNotFound(c, "location not found")
		}
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, &BotResponse{
		Bots: dto.BotsFromDomain(bots),
	})
}

func (h *BotHandler) Attack(c echo.Context) error {
	botSlug := c.Param("slug")
	if botSlug == "" {
		return ErrBadRequest(c, "bot slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	_, err = h.botService.Attack(c.Request().Context(), botSlug, userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrBotNotFound):
			return ErrNotFound(c, "bot not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrBadRequest(c, err.Error())
		}
	}

	return SuccessResponse(c, "attack initiated")
}
