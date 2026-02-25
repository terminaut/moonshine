-- +goose Up
-- +goose StatementBegin
UPDATE potions
SET image = image || '.png'
WHERE image IS NOT NULL
  AND image <> ''
  AND image !~ '\.[A-Za-z0-9]+$';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE potions
SET image = regexp_replace(image, '\.png$', '')
WHERE image LIKE 'potions/%'
  AND image ~ '\.png$';
-- +goose StatementEnd
