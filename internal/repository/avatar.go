package repository

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moonshine/internal/domain"
)

var (
	ErrAvatarNotFound = errors.New("avatar not found")
	ErrAvatarExists   = errors.New("avatar already exists")
)

type AvatarRepository struct {
	db *sqlx.DB
}

func NewAvatarRepository(db *sqlx.DB) *AvatarRepository {
	return &AvatarRepository{db: db}
}

func (r *AvatarRepository) Create(avatar *domain.Avatar) error {
	query := `
		INSERT INTO avatars (image, private)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(query,
		avatar.Image, avatar.Private,
	).Scan(&avatar.ID, &avatar.CreatedAt)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrAvatarExists
		}
		return err
	}
	return nil
}

func (r *AvatarRepository) FindByID(id uuid.UUID) (*domain.Avatar, error) {
	query := `
		SELECT id, created_at, deleted_at, image, private
		FROM avatars
		WHERE id = $1 AND deleted_at IS NULL
	`

	avatar := &domain.Avatar{}
	err := r.db.Get(avatar, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAvatarNotFound
		}
		return nil, err
	}

	return avatar, nil
}

func (r *AvatarRepository) FindByImage(image string) (*domain.Avatar, error) {
	query := `
		SELECT id, created_at, deleted_at, image, private
		FROM avatars
		WHERE image = $1 AND deleted_at IS NULL
	`

	avatar := &domain.Avatar{}
	err := r.db.Get(avatar, query, image)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAvatarNotFound
		}
		return nil, err
	}

	return avatar, nil
}

func (r *AvatarRepository) FindFirst() (*domain.Avatar, error) {
	query := `
		SELECT id, created_at, deleted_at, image, private
		FROM avatars
		WHERE deleted_at IS NULL
		LIMIT 1
	`

	avatar := &domain.Avatar{}
	err := r.db.Get(avatar, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAvatarNotFound
		}
		return nil, err
	}

	return avatar, nil
}

func (r *AvatarRepository) FindByIDs(ids []uuid.UUID) (map[uuid.UUID]*domain.Avatar, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.Avatar), nil
	}

	query, args, err := sqlx.In(`
		SELECT id, created_at, deleted_at, image, private
		FROM avatars
		WHERE id IN (?) AND deleted_at IS NULL
	`, UUIDsToStrings(ids))
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)
	var avatars []*domain.Avatar
	err = r.db.Select(&avatars, query, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*domain.Avatar, len(avatars))
	for _, avatar := range avatars {
		result[avatar.ID] = avatar
	}

	return result, nil
}

func (r *AvatarRepository) FindAll() ([]*domain.Avatar, error) {
	query := `
		SELECT id, created_at, deleted_at, image, private
		FROM avatars
		WHERE deleted_at IS NULL
		ORDER BY created_at ASC
	`

	var avatars []*domain.Avatar
	err := r.db.Select(&avatars, query)
	if err != nil {
		return nil, err
	}

	return avatars, nil
}
