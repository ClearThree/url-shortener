-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS short_url(
    id varchar NOT NULL,
    original_url varchar NOT NULL,
    created_at timestamp default NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE short_url;
-- +goose StatementEnd
