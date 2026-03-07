-- +goose Up
-- +goose StatementBegin
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    tool_category_id UUID NOT NULL,
    tool_item_id UUID NOT NULL,
    image VARCHAR(255),
    CONSTRAINT fk_resources_tool_category FOREIGN KEY (tool_category_id) REFERENCES tool_categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_resources_tool_item FOREIGN KEY (tool_item_id) REFERENCES tool_items(id) ON DELETE RESTRICT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS resources;
-- +goose StatementEnd
