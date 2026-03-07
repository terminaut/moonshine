-- +goose Up
-- +goose StatementBegin
CREATE TABLE location_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    location_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    CONSTRAINT fk_location_resources_location FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE,
    CONSTRAINT fk_location_resources_resource FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS location_resources;
-- +goose StatementEnd
