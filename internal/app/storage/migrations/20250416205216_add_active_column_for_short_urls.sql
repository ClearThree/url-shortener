-- +goose Up
-- +goose StatementBegin
ALTER TABLE "short_url" ADD COLUMN IF NOT EXISTS active BOOL DEFAULT true;
ALTER TABLE "short_url" ADD COLUMN IF NOT EXISTS modified_at timestamp DEFAULT NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "short_url" DROP COLUMN IF EXISTS modified_at;
ALTER TABLE "short_url" DROP COLUMN IF EXISTS active;
-- +goose StatementEnd
