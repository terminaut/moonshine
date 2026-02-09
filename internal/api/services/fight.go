package services

import (
	"context"
	"errors"
	"math"
	"math/rand"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
)

type FightService struct {
	fightRepo *repository.FightRepository
	botRepo   *repository.BotRepository
	userRepo  *repository.UserRepository
	roundRepo *repository.RoundRepository
	db        *sqlx.DB
}

func NewFightService(db *sqlx.DB) *FightService {
	return &FightService{
		fightRepo: repository.NewFightRepository(db),
		botRepo:   repository.NewBotRepository(db),
		userRepo:  repository.NewUserRepository(db),
		roundRepo: repository.NewRoundRepository(db),
		db:        db,
	}
}

type GetCurrentFightResult struct {
	User  *domain.User
	Bot   *domain.Bot
	Fight *domain.Fight
}

var ErrNoActiveFight = errors.New("no active fight")
var ErrUserNotFound = errors.New("user not found")
var ErrBotNotFound = errors.New("bot not found")
var ErrInvalidBodyPart = errors.New("invalid body part")

func isValidBodyPart(part string) bool {
	bodyPart := domain.BodyPart(part)
	for _, validPart := range domain.BodyParts {
		if validPart == bodyPart {
			return true
		}
	}
	return false
}

func (s *FightService) GetCurrentFight(ctx context.Context, userID uuid.UUID) (*GetCurrentFightResult, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	fight, err := s.fightRepo.FindActiveByUserID(userID)
	if err != nil {
		return nil, ErrNoActiveFight
	}

	rounds, err := s.roundRepo.FindByFightID(fight.ID)
	if err != nil {
		return nil, ErrNoActiveFight
	}
	fight.Rounds = rounds

	bot, err := s.botRepo.FindByID(fight.BotID)
	if err != nil {
		return nil, ErrBotNotFound
	}

	return &GetCurrentFightResult{
		User:  user,
		Bot:   bot,
		Fight: fight,
	}, nil
}

func (s *FightService) Hit(ctx context.Context, userID uuid.UUID, playerAttackPoint, playerDefensePoint string) (*GetCurrentFightResult, error) {
	if !isValidBodyPart(playerAttackPoint) {
		return nil, ErrInvalidBodyPart
	}
	if !isValidBodyPart(playerDefensePoint) {
		return nil, ErrInvalidBodyPart
	}

	fight, err := s.fightRepo.FindActiveByUserID(userID)
	if err != nil {
		return nil, ErrNoActiveFight
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	bot, err := s.botRepo.FindByID(fight.BotID)
	if err != nil {
		return nil, ErrBotNotFound
	}

	rounds, err := s.roundRepo.FindByFightID(fight.ID)
	if err != nil {
		return nil, ErrInternalError
	}

	if len(rounds) == 0 {
		return nil, ErrInternalError
	}

	currentRound := rounds[0]

	botAttackPoint := string(domain.BodyParts[rand.Intn(len(domain.BodyParts))])
	botDefensePoint := string(domain.BodyParts[rand.Intn(len(domain.BodyParts))])

	playerDmg := calculateDamage(user.Attack, bot.Defense, playerAttackPoint, botDefensePoint)
	botDmg := calculateDamage(bot.Attack, user.Defense, botAttackPoint, playerDefensePoint)

	finalPlayerHp := calculateFinalHp(currentRound.PlayerHp, botDmg)
	finalBotHp := calculateFinalHp(currentRound.BotHp, playerDmg)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, ErrInternalError
	}
	defer tx.Rollback()

	roundRepoTx := repository.NewRoundRepository(tx)
	fightRepoTx := repository.NewFightRepository(tx)

	if err = roundRepoTx.FinishRound(currentRound.ID, botAttackPoint, botDefensePoint, playerAttackPoint, playerDefensePoint,
		playerDmg, botDmg, finalPlayerHp, finalBotHp); err != nil {
		return nil, ErrInternalError
	}

	if finalPlayerHp == 0 || finalBotHp == 0 {
		fight.DroppedGold = calculateDroppedGold(bot.Level)
		fight.Exp = calculateExp(finalBotHp, user.Level, bot.Level)

		lvl := calculateLvl(user.Level, user.Exp, fight.Exp)

		if lvl > user.Level {
			user.CurrentHp = int(user.Hp)
		} else {
			user.CurrentHp = finalPlayerHp
		}

		if err = s.userRepo.UpdateWithExt(tx, userID, fight.DroppedGold, fight.Exp, lvl, user.CurrentHp); err != nil {
			return nil, ErrInternalError
		}

		finished, err := fightRepoTx.Finish(fight.ID, fight.DroppedGold, fight.Exp)
		if err != nil {
			return nil, ErrInternalError
		}
		fight = finished
	} else {
		if err = roundRepoTx.Create(fight.ID, finalPlayerHp, uint(finalBotHp)); err != nil {
			return nil, ErrInternalError
		}
	}

	updatedRounds, err := roundRepoTx.FindByFightID(fight.ID)
	if err != nil {
		return nil, ErrInternalError
	}
	fight.Rounds = updatedRounds

	if err = tx.Commit(); err != nil {
		return nil, ErrInternalError
	}

	return &GetCurrentFightResult{
		User:  user,
		Bot:   bot,
		Fight: fight,
	}, nil
}

func calculateDamage(attack, defense uint, attackPoint, defensePoint string) uint {
	var base int
	if attackPoint == defensePoint {
		base = int(attack) - int(defense)
	} else {
		base = int(attack)
	}
	if base <= 0 {
		return 0
	}
	mult := 0.9 + rand.Float64()*0.2
	dmg := int(math.Round(float64(base) * mult))
	if dmg < 0 {
		return 0
	}
	return uint(dmg)
}

func calculateFinalHp(currentHp int, damage uint) int {
	res := currentHp - int(damage)

	if res < 0 {
		return 0
	}
	return res
}

func calculateDroppedGold(botLvl uint) uint {
	limitDroppedGold := botLvl * 5

	if rand.Intn(3) == 1 {
		return uint(rand.Intn(int(limitDroppedGold)) + 1)
	}

	return 0
}

func calculateExp(botFinalHp int, playerLvl, botLvl uint) uint {
	if botFinalHp > 0 || playerLvl >= 20 {
		return 0
	}

	nextLevel := playerLvl + 1
	requiredExp, exists := domain.LevelMatrix[nextLevel]
	if !exists {
		return 0
	}

	bots := botsToLevel(playerLvl)
	baseExp := float64(requiredExp) / float64(bots)
	mod := levelModifier(playerLvl, botLvl)

	return uint(baseExp * mod)
}

func botsToLevel(playerLvl uint) uint {
	return uint(5 * math.Pow(1.6, float64(playerLvl-1)))
}

func levelModifier(playerLvl, botLvl uint) float64 {
	diff := int(botLvl) - int(playerLvl)

	switch {
	case diff == 0:
		return 1.0
	case diff > 0:
		return 1.0 + float64(diff)*0.25
	default:
		return 1.0 / (1.0 + float64(-diff)*0.5)
	}
}

func calculateLvl(playerLvl, currentExp, gotExp uint) uint {
	newExp := currentExp + gotExp
	newLevel := playerLvl

	for {
		nextLevel := newLevel + 1
		if nextLevel > 20 {
			break
		}

		requiredExp, exists := domain.LevelMatrix[nextLevel]
		if !exists {
			break
		}

		if newExp >= requiredExp {
			newLevel = nextLevel
		} else {
			break
		}
	}

	return newLevel
}
