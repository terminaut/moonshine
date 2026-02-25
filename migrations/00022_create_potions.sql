-- +goose Up
-- +goose StatementBegin
CREATE TYPE potion_stat_type AS ENUM ('CURRENT_HP', 'ATTACK', 'DEFENSE');

CREATE TABLE potions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    stat potion_stat_type NOT NULL,
    value INTEGER NOT NULL DEFAULT 0,
    price INTEGER NOT NULL DEFAULT 0,
    image VARCHAR(255)
);

CREATE UNIQUE INDEX idx_potions_slug_unique ON potions (slug) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS potions;
DROP TYPE IF EXISTS potion_stat_type;
-- +goose StatementEnd
