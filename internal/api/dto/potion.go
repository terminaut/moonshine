package dto

import (
	"time"

	"moonshine/internal/domain"
)

type Potion struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Stat      string    `json:"stat"`
	Value     int       `json:"value"`
	Price     int       `json:"price"`
	Image     string    `json:"image"`
	CreatedAt time.Time `json:"createdAt"`
}

func PotionFromDomain(potion *domain.Potion) *Potion {
	if potion == nil {
		return nil
	}
	return &Potion{
		ID:        potion.ID.String(),
		Name:      potion.Name,
		Slug:      potion.Slug,
		Stat:      potion.Stat,
		Value:     int(potion.Value),
		Price:     int(potion.Price),
		Image:     potion.Image,
		CreatedAt: potion.CreatedAt,
	}
}

func PotionsFromDomain(potions []*domain.Potion) []*Potion {
	result := make([]*Potion, len(potions))
	for i, potion := range potions {
		result[i] = PotionFromDomain(potion)
	}
	return result
}
