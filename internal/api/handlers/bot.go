package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type BotHandler struct {
	botService   *services.BotService
	fightChecker *FightChecker
}

type BotResponse struct {
	Bots []*dto.Bot `json:"bots"`
}

func NewBotHandler(botService *services.BotService, fightChecker *FightChecker) *BotHandler {
	return &BotHandler{
		botService:   botService,
		fightChecker: fightChecker,
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

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
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
