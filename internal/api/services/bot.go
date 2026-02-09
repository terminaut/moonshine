package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type BotService struct {
	locationRepo *repository.LocationRepository
	botRepo      *repository.BotRepository
	userRepo     *repository.UserRepository
	fightRepo    *repository.FightRepository
	roundRepo    *repository.RoundRepository
	db           *sqlx.DB
}

func NewBotService(db *sqlx.DB) *BotService {
	return &BotService{
		locationRepo: repository.NewLocationRepository(db),
		botRepo:      repository.NewBotRepository(db),
		userRepo:     repository.NewUserRepository(db),
		fightRepo:    repository.NewFightRepository(db),
		roundRepo:    repository.NewRoundRepository(db),
		db:           db,
	}
}

func (s *BotService) GetBotsByLocationSlug(locationSlug string) ([]*domain.Bot, error) {
	if locationSlug == "" {
		return nil, errors.New("location slug is required")
	}

	if locationSlug == domain.WaywardPinesSlug {
		cells, err := s.locationRepo.FindAllCells()
		if err != nil {
			return nil, err
		}

		botsMap := make(map[uuid.UUID]*domain.Bot)
		for _, cell := range cells {
			bots, err := s.botRepo.FindBotsByLocationID(cell.ID)
			if err != nil {
				continue
			}
			for _, bot := range bots {
				botsMap[bot.ID] = bot
			}
		}

		result := make([]*domain.Bot, 0, len(botsMap))
		for _, bot := range botsMap {
			result = append(result, bot)
		}
		return result, nil
	}

	location, err := s.locationRepo.FindBySlug(locationSlug)
	if err != nil {
		return nil, err
	}

	bots, err := s.botRepo.FindBotsByLocationID(location.ID)
	if err != nil {
		return nil, err
	}

	return bots, nil
}

type AttackResult struct {
	User *domain.User
	Bot  *domain.Bot
}

func (s *BotService) Attack(ctx context.Context, botSlug string, userID uuid.UUID) (*AttackResult, error) {
	if botSlug == "" {
		return nil, errors.New("bot slug is required")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	inFight, err := s.userRepo.InFight(userID)
	if err != nil || inFight {
		return nil, errors.New("user is already in fight")
	}

	bot, err := s.botRepo.FindBySlug(botSlug)
	if err != nil {
		return nil, err
	}

	exists, err := s.locationRepo.HasBot(user.LocationID, bot.ID)
	if err != nil {
		return nil, errors.New("error checking bot location")
	}
	if !exists {
		return nil, errors.New("bot is not in the same location as user")
	}

	fightID, err := s.fightRepo.Create(&domain.Fight{
		UserID: user.ID,
		BotID:  bot.ID,
	})
	if err != nil {
		return nil, err
	}

	err = s.roundRepo.Create(fightID, user.CurrentHp, bot.Hp)
	if err != nil {
		return nil, err
	}

	return &AttackResult{
		User: user,
		Bot:  bot,
	}, nil
}
