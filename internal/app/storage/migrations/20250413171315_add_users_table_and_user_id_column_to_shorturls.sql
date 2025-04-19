-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users(
                                        id uuid PRIMARY KEY,
                                        created_at timestamp default NOW()
);
ALTER TABLE "short_url" ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE "short_url" ADD CONSTRAINT user_id_foreign_key FOREIGN KEY (user_id) REFERENCES users(id);
CREATE INDEX IF NOT EXISTS short_urls_user_id_idx ON short_url USING HASH (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX "short_urls_user_id_idx";
ALTER TABLE "short_url" DROP CONSTRAINT IF EXISTS user_id_foreign_key;
ALTER TABLE "short_url" DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
