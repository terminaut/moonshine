package repository

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository struct {
	db           *sqlx.DB
	locationRepo *LocationRepository
	avatarRepo   *AvatarRepository
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db:           db,
		locationRepo: NewLocationRepository(db),
		avatarRepo:   NewAvatarRepository(db),
	}
}

func (r *UserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (
			username, email, password, name, avatar_id, location_id,
			attack, defense, current_hp, exp, free_stats, gold, hp, level
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query,
		user.Username, user.Email, user.Password, user.Name, user.AvatarID, user.LocationID,
		user.Attack, user.Defense, user.CurrentHp, user.Exp, user.FreeStats, user.Gold, user.Hp, user.Level,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT users.id, users.created_at, users.updated_at, users.deleted_at, users.username, users.email, users.password, users.name, 
			users.avatar_id, users.location_id, users.attack, users.defense, users.current_hp, users.exp,
			users.free_stats, users.gold, users.hp, users.level,
			users.chest_equipment_item_id, users.belt_equipment_item_id, users.head_equipment_item_id,
			users.neck_equipment_item_id, users.weapon_equipment_item_id, users.shield_equipment_item_id,
			users.legs_equipment_item_id, users.feet_equipment_item_id, users.arms_equipment_item_id,
			users.hands_equipment_item_id, users.ring1_equipment_item_id, users.ring2_equipment_item_id,
			users.ring3_equipment_item_id, users.ring4_equipment_item_id, avatars.image as avatar
		FROM users
		LEFT JOIN avatars ON avatars.id = users.avatar_id
		WHERE users.id = $1 AND users.deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.Get(user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	query := `
		SELECT users.id, users.created_at, users.updated_at, users.deleted_at, users.username, users.email, users.password, users.name, 
			users.avatar_id, users.location_id, users.attack, users.defense, users.current_hp, users.exp,
			users.free_stats, users.gold, users.hp, users.level,
			users.chest_equipment_item_id, users.belt_equipment_item_id, users.head_equipment_item_id,
			users.neck_equipment_item_id, users.weapon_equipment_item_id, users.shield_equipment_item_id,
			users.legs_equipment_item_id, users.feet_equipment_item_id, users.arms_equipment_item_id,
			users.hands_equipment_item_id, users.ring1_equipment_item_id, users.ring2_equipment_item_id,
			users.ring3_equipment_item_id, users.ring4_equipment_item_id, avatars.image as avatar
		FROM users
		LEFT JOIN avatars ON avatars.id = users.avatar_id
		WHERE users.username = $1 AND users.deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.Get(user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) UpdateGold(userID uuid.UUID, newGold uint) error {
	query := `UPDATE users SET gold = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.Exec(query, newGold, userID)
	return err
}

func (r *UserRepository) UpdateAvatarID(userID uuid.UUID, avatarID *uuid.UUID) error {
	query := `UPDATE users SET avatar_id = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.Exec(query, avatarID, userID)
	return err
}

type HPUpdate struct {
	UserID    uuid.UUID `db:"id"`
	CurrentHp int       `db:"current_hp"`
	Hp        uint      `db:"hp"`
}

func (r *UserRepository) RegenerateAllUsersHealth(percent float64) ([]HPUpdate, error) {
	query := `
		UPDATE users 
		SET current_hp = LEAST(
			current_hp + GREATEST(1, ROUND(hp * $1 / 100.0)), 
			hp
		)
		WHERE users.deleted_at IS NULL 
		    AND current_hp < hp
		    AND NOT EXISTS (
		        SELECT 1 FROM fights 
		        WHERE fights.user_id = users.id 
		        AND fights.status = $2
		        AND fights.deleted_at IS NULL
		    )
		RETURNING id, current_hp, hp
	`
	var updates []HPUpdate
	err := r.db.Select(&updates, query, percent, domain.FightStatusInProgress)
	if err != nil {
		return nil, err
	}
	return updates, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "duplicate key")
}

func (r *UserRepository) UpdateLocationID(userID uuid.UUID, locationID uuid.UUID) error {
	query := `UPDATE users SET location_id = $1 WHERE id = $2`
	_, err := r.db.Exec(query, locationID, userID)
	return err
}

func (r *UserRepository) Update(userID uuid.UUID, addedGold, addedExp, newLevel uint, newCurrentHp int) error {
	return r.UpdateWithExt(r.db, userID, addedGold, addedExp, newLevel, newCurrentHp)
}

func (r *UserRepository) UpdateWithExt(h ExtHandle, userID uuid.UUID, addedGold, addedExp, newLevel uint, newCurrentHp int) error {
	if newCurrentHp < 0 {
		newCurrentHp = 0
	}

	query := `
		UPDATE users
		SET gold = gold + $1,
		    exp = exp + $2,
		    level = $3,
		    current_hp = $4
		WHERE id = $5 AND deleted_at IS NULL
	`
	_, err := h.Exec(query, addedGold, addedExp, newLevel, newCurrentHp, userID)
	return err
}

func (r *UserRepository) InFight(userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM fights WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL)`

	exists := false
	err := r.db.Get(&exists, query, userID, domain.FightStatusInProgress)

	return exists, err
}

func (r *UserRepository) GetHPForUsers(userIDs []uuid.UUID) ([]HPUpdate, error) {
	if len(userIDs) == 0 {
		return []HPUpdate{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, current_hp, hp 
		FROM users 
		WHERE id IN (?) AND deleted_at IS NULL
	`, userIDs)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)
	var updates []HPUpdate
	err = r.db.Select(&updates, query, args...)
	if err != nil {
		return nil, err
	}
	return updates, nil
}
