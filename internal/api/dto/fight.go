package dto

import (
	"time"

	"moonshine/internal/domain"
)

type Round struct {
	ID                 string    `json:"id"`
	FightID            string    `json:"fightId"`
	PlayerDamage       int       `json:"playerDamage"`
	BotDamage          int       `json:"botDamage"`
	Status             string    `json:"status"`
	PlayerHp           int       `json:"playerHp"`
	BotHp              int       `json:"botHp"`
	PlayerAttackPoint  *string   `json:"playerAttackPoint,omitempty"`
	PlayerDefensePoint *string   `json:"playerDefensePoint,omitempty"`
	BotAttackPoint     *string   `json:"botAttackPoint,omitempty"`
	BotDefensePoint    *string   `json:"botDefensePoint,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

type Fight struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	BotID         string    `json:"botId"`
	Status        string    `json:"status"`
	DroppedGold   int       `json:"droppedGold"`
	Exp           int       `json:"exp"`
	DroppedItemID *string   `json:"droppedItemId,omitempty"`
	Rounds        []*Round  `json:"rounds"`
	CreatedAt     time.Time `json:"createdAt"`
}

func RoundFromDomain(round *domain.Round) *Round {
	if round == nil {
		return nil
	}

	result := &Round{
		ID:           round.ID.String(),
		FightID:      round.FightID.String(),
		PlayerDamage: int(round.PlayerDamage),
		BotDamage:    int(round.BotDamage),
		Status:       string(round.Status),
		PlayerHp:     round.PlayerHp,
		BotHp:        round.BotHp,
		CreatedAt:    round.CreatedAt,
	}

	if round.PlayerAttackPoint != nil {
		part := string(*round.PlayerAttackPoint)
		result.PlayerAttackPoint = &part
	}
	if round.PlayerDefensePoint != nil {
		part := string(*round.PlayerDefensePoint)
		result.PlayerDefensePoint = &part
	}
	if round.BotAttackPoint != nil {
		part := string(*round.BotAttackPoint)
		result.BotAttackPoint = &part
	}
	if round.BotDefensePoint != nil {
		part := string(*round.BotDefensePoint)
		result.BotDefensePoint = &part
	}

	return result
}

func RoundsFromDomain(rounds []*domain.Round) []*Round {
	result := make([]*Round, len(rounds))
	for i, round := range rounds {
		result[i] = RoundFromDomain(round)
	}
	return result
}

func FightFromDomain(fight *domain.Fight) *Fight {
	if fight == nil {
		return nil
	}

	result := &Fight{
		ID:          fight.ID.String(),
		UserID:      fight.UserID.String(),
		BotID:       fight.BotID.String(),
		Status:      string(fight.Status),
		DroppedGold: int(fight.DroppedGold),
		Exp:         int(fight.Exp),
		CreatedAt:   fight.CreatedAt,
		Rounds:      []*Round{},
	}

	if fight.Rounds != nil {
		result.Rounds = RoundsFromDomain(fight.Rounds)
	}

	if fight.DroppedItemID != nil {
		id := fight.DroppedItemID.String()
		result.DroppedItemID = &id
	}

	return result
}
