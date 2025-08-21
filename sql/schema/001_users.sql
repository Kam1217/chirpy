-- +goose Up
CREATE TABLE users (
    ID UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    eamil TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;