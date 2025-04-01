-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS short_url(
    id bigserial PRIMARY KEY,
    short_url text NOT NULL,
    original_url text NOT NULL,
    correlation_id text,
    created_at timestamp default NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE short_url;
-- +goose StatementEnd
