-- +goose Up
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_ring3_equipment;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_ring4_equipment;
ALTER TABLE users DROP COLUMN IF EXISTS ring3_equipment_item_id;
ALTER TABLE users DROP COLUMN IF EXISTS ring4_equipment_item_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS ring3_equipment_item_id UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ring4_equipment_item_id UUID;
ALTER TABLE users
	ADD CONSTRAINT fk_users_ring3_equipment FOREIGN KEY (ring3_equipment_item_id) REFERENCES equipment_items(id) ON DELETE SET NULL;
ALTER TABLE users
	ADD CONSTRAINT fk_users_ring4_equipment FOREIGN KEY (ring4_equipment_item_id) REFERENCES equipment_items(id) ON DELETE SET NULL;
-- +goose StatementEnd
