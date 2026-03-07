package handlers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type UserHandler struct {
	userService       *services.UserService
	inventoryService  *services.InventoryService
	userRepo          *repository.UserRepository
	equipmentItemRepo *repository.EquipmentItemRepository
	potionRepo        *repository.PotionRepository
	toolItemRepo      *repository.ToolItemRepository
	fightChecker      *FightChecker
}

func NewUserHandler(
	userService *services.UserService,
	inventoryService *services.InventoryService,
	userRepo *repository.UserRepository,
	equipmentItemRepo *repository.EquipmentItemRepository,
	potionRepo *repository.PotionRepository,
	toolItemRepo *repository.ToolItemRepository,
	fightChecker *FightChecker,
) *UserHandler {
	return &UserHandler{
		userService:       userService,
		inventoryService:  inventoryService,
		userRepo:          userRepo,
		equipmentItemRepo: equipmentItemRepo,
		potionRepo:        potionRepo,
		toolItemRepo:      toolItemRepo,
		fightChecker:      fightChecker,
	}
}

func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	user, location, inFight, err := h.userService.GetCurrentUserWithRelations(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrNotFound(c, "user not found")
		}
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.UserFromDomain(user, location, nil, inFight))
}

func (h *UserHandler) GetUserInventory(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	items, err := h.inventoryService.GetUserInventory(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	potions, err := h.inventoryService.GetUserInventoryPotions(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	toolItems, err := h.inventoryService.GetUserInventoryToolItems(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	resources, err := h.inventoryService.GetUserInventoryResources(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.InventoryResponse{
		EquipmentItems: dto.EquipmentItemsFromDomain(items),
		Potions:        dto.PotionsFromDomain(potions),
		ToolItems:      dto.ToolItemsFromDomain(toolItems),
		Resources:      dto.ResourcesFromDomain(resources),
	})
}

func (h *UserHandler) GetUserEquippedItems(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return ErrNotFound(c, "user not found")
	}

	slots := []struct {
		name string
		id   *uuid.UUID
	}{
		{"chest", user.ChestEquipmentItemID},
		{"belt", user.BeltEquipmentItemID},
		{"head", user.HeadEquipmentItemID},
		{"neck", user.NeckEquipmentItemID},
		{"weapon", user.WeaponEquipmentItemID},
		{"shield", user.ShieldEquipmentItemID},
		{"legs", user.LegsEquipmentItemID},
		{"feet", user.FeetEquipmentItemID},
		{"arms", user.ArmsEquipmentItemID},
		{"hands", user.HandsEquipmentItemID},
		{"ring1", user.Ring1EquipmentItemID},
		{"ring2", user.Ring2EquipmentItemID},
	}

	result := map[string]any{}

	var equipmentIDs []uuid.UUID
	for _, s := range slots {
		if s.id != nil {
			equipmentIDs = append(equipmentIDs, *s.id)
		}
	}

	if len(equipmentIDs) > 0 {
		list, err := h.equipmentItemRepo.FindByIDs(equipmentIDs)
		if err != nil {
			return ErrInternalServerError(c)
		}
		idToItem := make(map[uuid.UUID]*dto.EquipmentItem)
		for _, it := range list {
			idToItem[it.ID] = dto.EquipmentItemFromDomain(it)
		}
		for _, s := range slots {
			if s.id != nil {
				if d, ok := idToItem[*s.id]; ok {
					result[s.name] = d
				}
			}
		}
	}

	if user.ToolItemID != nil {
		toolItem, err := h.toolItemRepo.FindByID(*user.ToolItemID)
		if err == nil {
			result["tool"] = dto.ToolItemFromDomain(toolItem)
		}
	}

	potionSlots := []struct {
		name string
		id   *uuid.UUID
	}{
		{"potion1", user.Potion1ID},
		{"potion2", user.Potion2ID},
		{"potion3", user.Potion3ID},
	}

	var potionIDs []uuid.UUID
	for _, s := range potionSlots {
		if s.id != nil {
			potionIDs = append(potionIDs, *s.id)
		}
	}

	if len(potionIDs) > 0 {
		potions, err := h.potionRepo.FindByIDs(potionIDs)
		if err != nil {
			return ErrInternalServerError(c)
		}
		idToPotion := make(map[uuid.UUID]*dto.Potion)
		for _, p := range potions {
			idToPotion[p.ID] = dto.PotionFromDomain(p)
		}
		for _, s := range potionSlots {
			if s.id != nil {
				if d, ok := idToPotion[*s.id]; ok {
					result[s.name] = d
				}
			}
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (h *UserHandler) UpdateCurrentUser(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return ErrBadRequest(c, "invalid request")
	}

	var avatarID *uuid.UUID
	if req.AvatarID != nil {
		parsedID, err := uuid.Parse(*req.AvatarID)
		if err != nil {
			return ErrBadRequest(c, "invalid avatar ID")
		}
		avatarID = &parsedID
	}

	_, err = h.userService.UpdateUser(c.Request().Context(), userID, avatarID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		case errors.Is(err, repository.ErrAvatarNotFound):
			return ErrNotFound(c, "avatar not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	user, location, inFight, err := h.userService.GetCurrentUserWithRelations(c.Request().Context(), userID)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.UserFromDomain(user, location, nil, inFight))
}
