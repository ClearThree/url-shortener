-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS short_urls_original_url_udx ON short_url(original_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX short_urls_original_url_udx;
-- +goose StatementEnd
