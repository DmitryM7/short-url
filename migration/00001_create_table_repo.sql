-- +goose Up
-- +goose StatementBegin
CREATE TABLE repo (
                   "id" SERIAL PRIMARY KEY,
                   "shorturl" VARCHAR NOT NULL UNIQUE,
                   "url" VARCHAR NOT NULL UNIQUE
                   )
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE repo
-- +goose StatementEnd
