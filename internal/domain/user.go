package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Model
	UpdatedAt             time.Time  `db:"updated_at"`
	Attack                uint       `db:"attack"`
	AvatarID              *uuid.UUID `db:"avatar_id"`
	CurrentHp             int        `db:"current_hp"`
	Defense               uint       `db:"defense"`
	Email                 string     `db:"email"`
	Exp                   uint       `db:"exp"`
	FreeStats             uint       `db:"free_stats"`
	Gold                  uint       `db:"gold"`
	Hp                    uint       `db:"hp"`
	Level                 uint       `db:"level"`
	LocationID            uuid.UUID  `db:"location_id"`
	Name                  string     `db:"name"`
	Password              string     `db:"password"`
	Username              string     `db:"username"`
	ChestEquipmentItemID  *uuid.UUID `db:"chest_equipment_item_id"`
	BeltEquipmentItemID   *uuid.UUID `db:"belt_equipment_item_id"`
	HeadEquipmentItemID   *uuid.UUID `db:"head_equipment_item_id"`
	NeckEquipmentItemID   *uuid.UUID `db:"neck_equipment_item_id"`
	WeaponEquipmentItemID *uuid.UUID `db:"weapon_equipment_item_id"`
	ShieldEquipmentItemID *uuid.UUID `db:"shield_equipment_item_id"`
	LegsEquipmentItemID   *uuid.UUID `db:"legs_equipment_item_id"`
	FeetEquipmentItemID   *uuid.UUID `db:"feet_equipment_item_id"`
	ArmsEquipmentItemID   *uuid.UUID `db:"arms_equipment_item_id"`
	HandsEquipmentItemID  *uuid.UUID `db:"hands_equipment_item_id"`
	Ring1EquipmentItemID  *uuid.UUID `db:"ring1_equipment_item_id"`
	Ring2EquipmentItemID  *uuid.UUID `db:"ring2_equipment_item_id"`
	Ring3EquipmentItemID  *uuid.UUID `db:"ring3_equipment_item_id"`
	Ring4EquipmentItemID  *uuid.UUID `db:"ring4_equipment_item_id"`
	Avatar                string     `db:"avatar"`
}

var LevelMatrix = map[uint]uint{
	1:  0,
	2:  100,
	3:  200,
	4:  400,
	5:  800,
	6:  1500,
	7:  3000,
	8:  5000,
	9:  10000,
	10: 15000,
	11: 20000,
	12: 25000,
	13: 30000,
	14: 35000,
	15: 40000,
	16: 45000,
	17: 50000,
	18: 55000,
	19: 60000,
	20: 65000,
}

func (user *User) ReachedNewLevel() bool {
	nextLevel := user.Level + 1
	requiredExp, exists := LevelMatrix[nextLevel]
	if !exists {
		return false
	}
	return user.Exp >= requiredExp
}

func (user *User) RegenerateHealth(percent float64) int {
	// Ensure current HP is at least 0
	if user.CurrentHp < 0 {
		user.CurrentHp = 0
	}

	maxHp := int(user.Hp)
	if user.CurrentHp >= maxHp {
		return maxHp
	}

	regeneration := int(float64(user.Hp) * percent / 100.0)

	minRegeneration := 5
	if regeneration < minRegeneration {
		regeneration = minRegeneration
	}

	newHp := user.CurrentHp + regeneration

	if newHp > maxHp {
		return maxHp
	}

	return newHp
}
