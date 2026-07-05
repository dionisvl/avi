-- +goose Up
-- +goose StatementBegin

ALTER TYPE payment_purpose ADD VALUE IF NOT EXISTS 'demo_checkout';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- PostgreSQL cannot remove enum values without recreating the type. Keep this
-- migration irreversible; the value is harmless for older rows.

-- +goose StatementEnd
