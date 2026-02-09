package domain

import "github.com/google/uuid"

type BodyPart string

const (
	BodyPartHead  BodyPart = "HEAD"
	BodyPartNeck  BodyPart = "NECK"
	BodyPartChest BodyPart = "CHEST"
	BodyPartBelt  BodyPart = "BELT"
	BodyPartLegs  BodyPart = "LEGS"
)

var BodyParts = []BodyPart{
	BodyPartHead,
	BodyPartNeck,
	BodyPartChest,
	BodyPartBelt,
	BodyPartLegs,
}

type RoundStatus string

const (
	RoundStatusInProgress RoundStatus = "IN_PROGRESS"
	RoundStatusFinished   RoundStatus = "FINISHED"
)

type Round struct {
	Model
	FightID            uuid.UUID   `db:"fight_id"`
	PlayerDamage       uint        `db:"player_damage"`
	BotDamage          uint        `db:"bot_damage"`
	Status             RoundStatus `db:"status"`
	PlayerHp           int         `db:"player_hp"`
	BotHp              int         `db:"bot_hp"`
	PlayerAttackPoint  *BodyPart   `db:"player_attack_point"`
	PlayerDefensePoint *BodyPart   `db:"player_defense_point"`
	BotAttackPoint     *BodyPart   `db:"bot_attack_point"`
	BotDefensePoint    *BodyPart   `db:"bot_defense_point"`
}
