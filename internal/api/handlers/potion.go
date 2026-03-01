package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type PotionHandler struct {
	potionRepo           *repository.PotionRepository
	potionBuyService     *services.PotionBuyService
	potionSellService    *services.PotionSellService
	potionTakeOnService  *services.PotionTakeOnService
	potionTakeOffService *services.PotionTakeOffService
	potionUseService     *services.PotionUseService
	userCache            r.Cache[domain.User]
	fightChecker         *FightChecker
}

func NewPotionHandler(
	potionBuyService *services.PotionBuyService,
	potionSellService *services.PotionSellService,
	potionTakeOnService *services.PotionTakeOnService,
	potionTakeOffService *services.PotionTakeOffService,
	potionUseService *services.PotionUseService,
	potionRepo *repository.PotionRepository,
	userCache r.Cache[domain.User],
	fightChecker *FightChecker,
) *PotionHandler {
	return &PotionHandler{
		potionRepo:           potionRepo,
		potionBuyService:     potionBuyService,
		potionSellService:    potionSellService,
		potionTakeOnService:  potionTakeOnService,
		potionTakeOffService: potionTakeOffService,
		potionUseService:     potionUseService,
		userCache:            userCache,
		fightChecker:         fightChecker,
	}
}

func (h *PotionHandler) invalidateUserCache(ctx context.Context, userID string) {
	if h.userCache != nil {
		_ = h.userCache.Delete(ctx, userID)
	}
}

func (h *PotionHandler) GetPotions(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	potions, err := h.potionRepo.FindAll()
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.PotionsFromDomain(potions))
}

func (h *PotionHandler) BuyPotion(c echo.Context) error {
	potionSlug := c.Param("slug")
	if potionSlug == "" {
		return ErrBadRequest(c, "potion slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.potionBuyService.BuyPotion(c.Request().Context(), userID, potionSlug)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrPotionNotFound):
			return ErrNotFound(c, "potion not found")
		case errors.Is(err, services.ErrInsufficientGold):
			return ErrBadRequest(c, "insufficient gold")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "potion purchased successfully")
}

func (h *PotionHandler) SellPotion(c echo.Context) error {
	potionSlug := c.Param("slug")
	if potionSlug == "" {
		return ErrBadRequest(c, "potion slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.potionSellService.SellPotion(c.Request().Context(), userID, potionSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotOwned):
			return ErrBadRequest(c, "potion not owned")
		case errors.Is(err, repository.ErrPotionNotFound):
			return ErrNotFound(c, "potion not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "potion sold successfully")
}

func (h *PotionHandler) TakeOnPotion(c echo.Context) error {
	potionSlug := c.Param("slug")
	if potionSlug == "" {
		return ErrBadRequest(c, "potion slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	potion, err := h.potionRepo.FindBySlug(potionSlug)
	if err != nil {
		return ErrNotFound(c, "potion not found")
	}

	err = h.potionTakeOnService.TakeOnPotion(c.Request().Context(), userID, potion.ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrPotionNotFound):
			return ErrNotFound(c, "potion not found")
		case errors.Is(err, services.ErrItemNotInInventory):
			return ErrBadRequest(c, "potion not in inventory")
		case errors.Is(err, services.ErrInvalidEquipmentType):
			return ErrBadRequest(c, "invalid potion type")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "potion equipped successfully")
}

func (h *PotionHandler) TakeOffPotion(c echo.Context) error {
	slotName := c.Param("slot")
	if slotName == "" {
		return ErrBadRequest(c, "slot name is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.potionTakeOffService.TakeOffPotion(c.Request().Context(), userID, slotName)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoItemEquipped):
			return ErrBadRequest(c, "no potion equipped in this slot")
		case errors.Is(err, services.ErrInvalidEquipmentType):
			return ErrBadRequest(c, "invalid slot name")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "potion removed successfully")
}

func (h *PotionHandler) UsePotion(c echo.Context) error {
	slot := c.Param("slot")
	if slot == "" {
		return ErrBadRequest(c, "slot is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckInFight(c, userID); err != nil {
		return err
	}

	err = h.potionUseService.UsePotion(c.Request().Context(), userID, slot)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoItemEquipped):
			return ErrBadRequest(c, "no potion equipped in this slot")
		case errors.Is(err, services.ErrInvalidEquipmentType):
			return ErrBadRequest(c, "invalid slot")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())

	return SuccessResponse(c, "potion has been taken")
}
