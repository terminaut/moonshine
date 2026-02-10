package dto

import (
	"time"

	"github.com/google/uuid"

	"moonshine/internal/domain"
)

type User struct {
	ID                    string    `json:"id"`
	Username              string    `json:"username"`
	Email                 string    `json:"email"`
	Hp                    int       `json:"hp"`
	CurrentHp             int       `json:"currentHp"`
	Attack                int       `json:"attack"`
	Defense               int       `json:"defense"`
	Level                 int       `json:"level"`
	Gold                  int       `json:"gold"`
	Exp                   int       `json:"exp"`
	FreeStats             int       `json:"freeStats"`
	CreatedAt             time.Time `json:"createdAt"`
	Avatar                string    `json:"avatar"`
	ChestEquipmentItemID  *string   `json:"chestEquipmentItemId,omitempty"`
	BeltEquipmentItemID   *string   `json:"beltEquipmentItemId,omitempty"`
	HeadEquipmentItemID   *string   `json:"headEquipmentItemId,omitempty"`
	NeckEquipmentItemID   *string   `json:"neckEquipmentItemId,omitempty"`
	WeaponEquipmentItemID *string   `json:"weaponEquipmentItemId,omitempty"`
	ShieldEquipmentItemID *string   `json:"shieldEquipmentItemId,omitempty"`
	LegsEquipmentItemID   *string   `json:"legsEquipmentItemId,omitempty"`
	FeetEquipmentItemID   *string   `json:"feetEquipmentItemId,omitempty"`
	ArmsEquipmentItemID   *string   `json:"armsEquipmentItemId,omitempty"`
	HandsEquipmentItemID  *string   `json:"handsEquipmentItemId,omitempty"`
	Ring1EquipmentItemID  *string   `json:"ring1EquipmentItemId,omitempty"`
	Ring2EquipmentItemID  *string   `json:"ring2EquipmentItemId,omitempty"`
	Ring3EquipmentItemID  *string   `json:"ring3EquipmentItemId,omitempty"`
	Ring4EquipmentItemID  *string   `json:"ring4EquipmentItemId,omitempty"`
	LocationSlug          *string   `json:"locationSlug,omitempty"`
	Location              *Location `json:"location,omitempty"`
	InFight               bool      `json:"inFight"`
}

type Location struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Bots []*Bot `json:"bots"`
}

type Avatar struct {
	ID      string `json:"id"`
	Image   string `json:"image"`
	Private bool   `json:"private"`
}

func UserFromDomain(user *domain.User, location *domain.Location, bots []*domain.Bot, inFight bool) *User {
	if user == nil {
		return nil
	}

	result := &User{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Hp:        int(user.Hp),
		CurrentHp: int(user.CurrentHp),
		Attack:    int(user.Attack),
		Defense:   int(user.Defense),
		Level:     int(user.Level),
		Gold:      int(user.Gold),
		Exp:       int(user.Exp),
		FreeStats: int(user.FreeStats),
		CreatedAt: user.CreatedAt,
		InFight:   inFight,
		Avatar:    user.Avatar,
	}

	equipmentItemFields := []struct {
		source *uuid.UUID
		target **string
	}{
		{user.ChestEquipmentItemID, &result.ChestEquipmentItemID},
		{user.BeltEquipmentItemID, &result.BeltEquipmentItemID},
		{user.HeadEquipmentItemID, &result.HeadEquipmentItemID},
		{user.NeckEquipmentItemID, &result.NeckEquipmentItemID},
		{user.WeaponEquipmentItemID, &result.WeaponEquipmentItemID},
		{user.ShieldEquipmentItemID, &result.ShieldEquipmentItemID},
		{user.LegsEquipmentItemID, &result.LegsEquipmentItemID},
		{user.FeetEquipmentItemID, &result.FeetEquipmentItemID},
		{user.ArmsEquipmentItemID, &result.ArmsEquipmentItemID},
		{user.HandsEquipmentItemID, &result.HandsEquipmentItemID},
		{user.Ring1EquipmentItemID, &result.Ring1EquipmentItemID},
		{user.Ring2EquipmentItemID, &result.Ring2EquipmentItemID},
		{user.Ring3EquipmentItemID, &result.Ring3EquipmentItemID},
		{user.Ring4EquipmentItemID, &result.Ring4EquipmentItemID},
	}

	for _, field := range equipmentItemFields {
		if field.source != nil {
			id := field.source.String()
			*field.target = &id
		}
	}

	if location != nil && location.Slug != "" {
		result.LocationSlug = &location.Slug
		result.Location = &Location{
			ID:   location.ID.String(),
			Name: location.Name,
			Slug: location.Slug,
			Bots: BotsFromDomain(bots),
		}
	}

	return result
}

type UpdateUserRequest struct {
	AvatarID *string `json:"avatarId,omitempty"`
}

func AvatarFromDomain(avatar *domain.Avatar) *Avatar {
	if avatar == nil {
		return nil
	}
	return &Avatar{
		ID:      avatar.ID.String(),
		Image:   avatar.Image,
		Private: avatar.Private,
	}
}

func AvatarsFromDomain(avatars []*domain.Avatar) []*Avatar {
	result := make([]*Avatar, len(avatars))
	for i, avatar := range avatars {
		result[i] = AvatarFromDomain(avatar)
	}
	return result
}
