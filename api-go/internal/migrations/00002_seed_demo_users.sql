-- +goose Up
-- +goose StatementBegin

-- Seed demo users for the public demo build.
-- Passwords equal the email prefix so they can be displayed publicly on the login screen.
-- Bcrypt hashes (cost 10) are hardcoded and verified; idempotent via ON CONFLICT.
--   demo1@avi.test / demo1
--   demo2@avi.test / demo2
--   demo3@avi.test / demo3
INSERT INTO users (id, email, password_hash, roles, is_email_verified, name, locale, created_at, updated_at)
VALUES
    (
        '00000000-0000-0000-0009-000000000001',
        'demo1@avi.test',
        '$2a$10$HP1L1YWy6jN2iIQ.NWWGye0X9i4Sl2pz1LquILxjvbL/qriSqljVS',
        array['ROLE_USER']::text[],
        true,
        'Demo One',
        'en',
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0009-000000000002',
        'demo2@avi.test',
        '$2a$10$jkinWoVEgm/8UKNICc29mebvHz/13kWgAXNFDxHoj8AQ61TANogGm',
        array['ROLE_USER']::text[],
        true,
        'Demo Two',
        'en',
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0009-000000000003',
        'demo3@avi.test',
        '$2a$10$1LSaG2sXr/gZIz2XYEFR6.CS0wyY.0WBdIijzqXS2vxFdTsHOTgJa',
        array['ROLE_USER']::text[],
        true,
        'Demo Three',
        'en',
        NOW(),
        NOW()
    )
ON CONFLICT (email) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM users WHERE email IN ('demo1@avi.test', 'demo2@avi.test', 'demo3@avi.test');

-- +goose StatementEnd
