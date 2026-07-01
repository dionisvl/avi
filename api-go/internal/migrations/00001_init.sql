-- +goose Up
-- +goose StatementBegin

-- Item condition enum: new or used
CREATE TYPE item_condition AS ENUM ('new', 'used');

-- Item status enum: published, draft, archived, or sold
CREATE TYPE item_status AS ENUM ('published', 'draft', 'archived', 'sold');

-- Payment-related enums
CREATE TYPE payment_provider AS ENUM ('yookassa');
CREATE TYPE payment_purpose AS ENUM (
  'promote_listing',
  'listing_placement',
  'listing_boost',
  'subscription'
);
CREATE TYPE payment_status AS ENUM (
  'pending',
  'succeeded',
  'canceled',
  'refunded',
  'partially_refunded'
);
CREATE TYPE payment_event_status AS ENUM (
  'pending',
  'processed',
  'ignored',
  'failed'
);

-- Users table: core auth and profile
CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    email                   TEXT NOT NULL UNIQUE,
    password_hash           TEXT NOT NULL,
    roles                   TEXT[] NOT NULL DEFAULT array['ROLE_USER']::text[],
    token_version           INTEGER NOT NULL DEFAULT 1,
    is_email_verified       BOOLEAN NOT NULL DEFAULT FALSE,
    email_verify_code       TEXT,
    email_verify_code_expiry TIMESTAMPTZ,
    reset_code              TEXT,
    reset_code_expiry       TIMESTAMPTZ,
    name                    TEXT,
    locale                  VARCHAR(2) NOT NULL DEFAULT 'en',
    preferences             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_roles_gin ON users USING GIN (roles);
ALTER TABLE users ADD CONSTRAINT users_locale_check CHECK (locale IN ('ru', 'en'));

-- Cities reference table
-- slug: URL-safe latin identifier ("moscow", "saint-petersburg").
-- geoname_id: GeoNames ID (geonames.org) for unambiguous international identification.
-- names: JSONB map of locale -> display name, e.g. {"en": "New York", "ru": "Нью-Йорк"}.
-- is_active: false = city exists in DB (for existing data) but is hidden from selection UI.
CREATE TABLE cities (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    slug       TEXT NOT NULL UNIQUE,
    geoname_id INTEGER UNIQUE,
    names      JSONB NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    population INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_cities_slug ON cities(slug);
CREATE INDEX idx_cities_is_active ON cities(is_active);

-- Seed initial cities
INSERT INTO cities (id, slug, geoname_id, names, is_active, population) VALUES
    ('00000000-0000-0000-0001-000000000001', 'new-york',     5128581, '{"en": "New York", "ru": "Нью-Йорк"}',         TRUE, 8804190),
    ('00000000-0000-0000-0001-000000000002', 'london',       2643743, '{"en": "London", "ru": "Лондон"}',             TRUE, 8961989),
    ('00000000-0000-0000-0001-000000000003', 'los-angeles',  5368361, '{"en": "Los Angeles", "ru": "Лос-Анджелес"}',  TRUE, 3898747),
    ('00000000-0000-0000-0001-000000000004', 'toronto',      6167865, '{"en": "Toronto", "ru": "Торонто"}',           TRUE, 2794356),
    ('00000000-0000-0000-0001-000000000005', 'chicago',      4887398, '{"en": "Chicago", "ru": "Чикаго"}',            TRUE, 2746388),
    ('00000000-0000-0000-0001-000000000006', 'berlin',       2950159, '{"en": "Berlin", "ru": "Берлин"}',             TRUE, 3644826),
    ('00000000-0000-0000-0001-000000000007', 'madrid',       3117735, '{"en": "Madrid", "ru": "Мадрид"}',             TRUE, 3223334),
    ('00000000-0000-0000-0001-000000000008', 'paris',        2988507, '{"en": "Paris", "ru": "Париж"}',               TRUE, 2138551),
    ('00000000-0000-0000-0001-000000000009', 'sydney',       2147714, '{"en": "Sydney", "ru": "Сидней"}',             TRUE, 5312163),
    ('00000000-0000-0000-0001-000000000010', 'amsterdam',    2759794, '{"en": "Amsterdam", "ru": "Амстердам"}',       TRUE,  872680),
    ('00000000-0000-0000-0001-000000000011', 'dublin',       2964574, '{"en": "Dublin", "ru": "Дублин"}',             TRUE,  554554),
    ('00000000-0000-0000-0001-000000000012', 'vienna',       2761369, '{"en": "Vienna", "ru": "Вена"}',               TRUE, 1897491),
    ('00000000-0000-0000-0001-000000000013', 'warsaw',       756135,  '{"en": "Warsaw", "ru": "Варшава"}',            TRUE, 1790658),
    ('00000000-0000-0000-0001-000000000014', 'lisbon',       2267057, '{"en": "Lisbon", "ru": "Лиссабон"}',           TRUE,  544851),
    ('00000000-0000-0000-0001-000000000015', 'stockholm',    2673730, '{"en": "Stockholm", "ru": "Стокгольм"}',       TRUE,  975551);

-- Categories reference table (replaces breeds)
-- Hierarchical: parent_id can be NULL for top-level categories
-- Seed includes 8 major classifieds categories
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    slug        TEXT NOT NULL UNIQUE,
    parent_id   UUID REFERENCES categories(id),
    names       JSONB NOT NULL,
    sort_order  SMALLINT NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_categories_slug ON categories(slug);
CREATE INDEX idx_categories_parent_id ON categories(parent_id);

-- Seed 8 top-level categories
INSERT INTO categories (id, slug, parent_id, names, sort_order, is_active) VALUES
    ('00000000-0000-0000-0003-000000000001', 'electronics',   NULL, '{"en": "Electronics", "ru": "Электроника"}',           1, TRUE),
    ('00000000-0000-0000-0003-000000000002', 'transport',     NULL, '{"en": "Transport", "ru": "Транспорт"}',               2, TRUE),
    ('00000000-0000-0000-0003-000000000003', 'real-estate',   NULL, '{"en": "Real Estate", "ru": "Недвижимость"}',          3, TRUE),
    ('00000000-0000-0000-0003-000000000004', 'clothing',      NULL, '{"en": "Clothing", "ru": "Одежда"}',                   4, TRUE),
    ('00000000-0000-0000-0003-000000000005', 'home-garden',   NULL, '{"en": "Home & Garden", "ru": "Дом и сад"}',           5, TRUE),
    ('00000000-0000-0000-0003-000000000006', 'hobbies',       NULL, '{"en": "Hobbies & Leisure", "ru": "Хобби и отдых"}',   6, TRUE),
    ('00000000-0000-0000-0003-000000000007', 'jobs',          NULL, '{"en": "Jobs", "ru": "Работа"}',                       7, TRUE),
    ('00000000-0000-0000-0003-000000000008', 'animals',       NULL, '{"en": "Animals", "ru": "Животные"}',                  8, TRUE);

-- Items table: classifieds listings (replaces animals)
CREATE TABLE items (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    seller_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    category_id     UUID NOT NULL REFERENCES categories(id),
    city_id         UUID NOT NULL REFERENCES cities(id),
    slug            TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    price_amount    BIGINT,
    price_currency  CHAR(3),
    condition       item_condition NOT NULL DEFAULT 'used',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    status          item_status NOT NULL DEFAULT 'published',
    thumbnail_id    UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_seller_id ON items(seller_id);
CREATE INDEX idx_items_category_id ON items(category_id);
CREATE INDEX idx_items_city_id ON items(city_id);
CREATE INDEX idx_items_status ON items(status);
CREATE INDEX idx_items_tags_gin ON items USING GIN (tags);
CREATE UNIQUE INDEX idx_items_slug ON items(slug) WHERE slug IS NOT NULL;
CREATE INDEX idx_items_catalog ON items(status, category_id, city_id);

-- Add price completeness constraint: both null or both set
ALTER TABLE items
    ADD CONSTRAINT items_price_complete
        CHECK ((price_amount IS NULL) = (price_currency IS NULL)),
    ADD CONSTRAINT items_price_non_negative
        CHECK (price_amount IS NULL OR price_amount >= 0);

-- Item photos (replaces animal_photos)
CREATE TABLE item_photos (
    id                UUID PRIMARY KEY DEFAULT uuidv7(),
    -- nullable: photos can be uploaded before an item exists and linked later
    item_id           UUID REFERENCES items(id) ON DELETE CASCADE,
    bucket            TEXT NOT NULL,
    object_key        TEXT NOT NULL,
    mime_type         TEXT NOT NULL,
    size_bytes        BIGINT NOT NULL,
    original_filename TEXT NOT NULL,
    sort_order        SMALLINT NOT NULL DEFAULT 0,
    uploader_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_item_photos_item_id ON item_photos(item_id);

-- Wire thumbnail_id FK on items to reference item_photos
ALTER TABLE items
    ADD CONSTRAINT items_thumbnail_id_fkey
        FOREIGN KEY (thumbnail_id) REFERENCES item_photos(id) ON DELETE SET NULL;

CREATE INDEX idx_items_thumbnail_id ON items(thumbnail_id);

-- User avatars
CREATE TABLE user_avatars (
    id                UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id           UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    bucket            TEXT NOT NULL,
    object_key        TEXT NOT NULL,
    mime_type         TEXT NOT NULL,
    size_bytes        BIGINT NOT NULL,
    original_filename TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User favorites (bookmarks)
CREATE TABLE user_favorites (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id    UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, item_id)
);

CREATE INDEX idx_user_favorites_user_id ON user_favorites(user_id);

-- Refresh sessions for JWT refresh token management
CREATE TABLE refresh_sessions (
    jti        UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_refresh_sessions_user_id ON refresh_sessions(user_id);
CREATE INDEX idx_refresh_sessions_expires_at ON refresh_sessions(expires_at);

-- Conversations and messages for inter-user chat about listings
CREATE TABLE conversations (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    user_a          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (user_a < user_b),
    UNIQUE (user_a, user_b)
);

CREATE INDEX idx_conversations_user_a ON conversations(user_a, last_message_at DESC);
CREATE INDEX idx_conversations_user_b ON conversations(user_b, last_message_at DESC);

CREATE TABLE chat_messages (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    conversation_id       UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body                  TEXT,
    attachment_object_key TEXT,
    attachment_mime       TEXT,
    attachment_size       BIGINT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (body IS NOT NULL OR attachment_object_key IS NOT NULL)
);

CREATE INDEX idx_chat_messages_conversation ON chat_messages(conversation_id, created_at DESC);

CREATE TABLE chat_reads (
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, user_id)
);

-- Payments for paid services (listing promotion, placement, boost, subscription, etc.).
-- subject_id conceptually references the item being promoted.
CREATE TABLE payments (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose payment_purpose NOT NULL,
  subject_id UUID NOT NULL,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL DEFAULT 'RUB',
  status payment_status NOT NULL DEFAULT 'pending',
  provider payment_provider NOT NULL DEFAULT 'yookassa',
  provider_payment_id TEXT,
  confirmation_url TEXT,
  idempotency_key TEXT NOT NULL,
  receipt JSONB,
  provider_method_id TEXT,
  provider_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  paid_at TIMESTAMPTZ,
  canceled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX payments_idempotency_key_uidx ON payments (idempotency_key);
CREATE UNIQUE INDEX payments_provider_payment_id_uidx
  ON payments (provider, provider_payment_id)
  WHERE provider_payment_id IS NOT NULL;
CREATE INDEX payments_user_id_idx ON payments (user_id);
CREATE UNIQUE INDEX payments_one_pending_per_subject_uidx
  ON payments (user_id, purpose, subject_id)
  WHERE status = 'pending';

CREATE TABLE payment_events (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  provider payment_provider NOT NULL,
  event_type TEXT NOT NULL,
  provider_payment_id TEXT NOT NULL,
  event_key TEXT NOT NULL,
  status payment_event_status NOT NULL DEFAULT 'pending',
  payload JSONB NOT NULL,
  error_message TEXT,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX payment_events_event_key_uidx
  ON payment_events (provider, event_key);
CREATE INDEX payment_events_provider_payment_id_idx
  ON payment_events (provider, provider_payment_id);
CREATE INDEX payment_events_status_idx ON payment_events (status);

CREATE TABLE paid_entitlements (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose payment_purpose NOT NULL,
  subject_id UUID NOT NULL,
  payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
  starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX paid_entitlements_active_uidx
  ON paid_entitlements (user_id, purpose, subject_id)
  WHERE expires_at IS NULL;
CREATE INDEX paid_entitlements_payment_id_idx ON paid_entitlements (payment_id);
CREATE INDEX paid_entitlements_subject_idx ON paid_entitlements (purpose, subject_id);

-- Seed test user for development
-- Email: test@example.com
-- Password: password123
INSERT INTO users (email, password_hash, roles, is_email_verified, created_at, updated_at)
VALUES (
    'test@example.com',
    '$2a$10$YyWORGOG7FhdUuTEs3TvZ.s/3hO96iUlo4aOrp./j7JC6VfczXrw2',
    array['ROLE_USER']::text[],
    true,
    NOW(),
    NOW()
);

-- Seed demo items across categories and cities
-- Item IDs with stable UUIDs
INSERT INTO items (id, seller_id, category_id, city_id, slug, title, description, price_amount, price_currency, condition, status, created_at, updated_at)
VALUES
    ('00000000-0000-0000-0004-000000000001',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000001',
     '00000000-0000-0000-0001-000000000001',
     'iphone-14-pro-max',
     'iPhone 14 Pro Max 256GB',
     'Excellent condition, 1 year of use, with the original box',
     89900,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000002',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000002',
     '00000000-0000-0000-0001-000000000002',
     'bmw-x5-2018',
     'BMW X5 2018',
     'Black SUV, excellent mechanical condition, single owner',
     3200000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000003',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000003',
     '00000000-0000-0000-0001-000000000001',
     'apartment-rent-new-york-center',
     'Studio apartment in downtown New York',
     'Cozy apartment with a park view, close to the subway, fully furnished',
     250000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000004',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000004',
     '00000000-0000-0000-0001-000000000002',
     'winter-jacket-xl',
     'Men''s winter jacket, size XL',
     'Brand new jacket, genuine down, never worn. Color: black',
     12000,
     'USD',
     'new',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000005',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000005',
     '00000000-0000-0000-0001-000000000001',
     'dining-table-solid-wood',
     'Solid wood dining table',
     'Beautiful 6-seat table, 1.8 m long, walnut, excellent condition',
     40000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000006',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000006',
     '00000000-0000-0000-0001-000000000003',
     'mountain-bike-trek',
     'Trek X-Caliber 29 mountain bike',
     'Bike in excellent condition, new brakes, size L',
     85000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000007',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000007',
     '00000000-0000-0000-0001-000000000001',
     'vacancy-senior-developer',
     'Job: Senior Backend Developer',
     'Looking for an experienced Go developer, salary $90-110k. Remote.',
     NULL,
     NULL,
     'new',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-0004-000000000008',
     (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1),
     '00000000-0000-0000-0003-000000000008',
     '00000000-0000-0000-0001-000000000001',
     'aquarium-200l-with-equipment',
     '200L aquarium with equipment',
     '200-liter aquarium, includes filter, heater and lighting. Used, in excellent condition',
     32000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW());

-- Seed demo item photos (2-3 per item) using external category placeholder images.
-- object_key holds an absolute URL, so query/item.MapItem returns it as-is
-- (see internal/platform/media.URL). bucket='external' marks the source.
-- Category mode (/photo/category/<cat>/W/H.webp) is deterministic per full URL:
-- the same URL always returns the same image, and a different ?n= value yields a
-- different image from that category. So ?n= keeps photos in a gallery distinct
-- and reproducible. WebP format.
INSERT INTO item_photos (id, item_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, uploader_id) VALUES
    -- iPhone (3 photos) — technology
    ('00000000-0000-0000-0005-000000000101', '00000000-0000-0000-0004-000000000001', 'external', 'https://placeholdpicsum.dev/photo/category/technology/800/600.webp?n=iphone-1', 'image/webp', 120000, 'iphone-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000102', '00000000-0000-0000-0004-000000000001', 'external', 'https://placeholdpicsum.dev/photo/category/technology/800/600.webp?n=iphone-2', 'image/webp', 120000, 'iphone-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000103', '00000000-0000-0000-0004-000000000001', 'external', 'https://placeholdpicsum.dev/photo/category/technology/800/600.webp?n=iphone-3', 'image/webp', 120000, 'iphone-3.webp', 2, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- BMW X5 (3 photos) — travel
    ('00000000-0000-0000-0005-000000000201', '00000000-0000-0000-0004-000000000002', 'external', 'https://placeholdpicsum.dev/photo/category/travel/800/600.webp?n=bmw-1', 'image/webp', 120000, 'bmw-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000202', '00000000-0000-0000-0004-000000000002', 'external', 'https://placeholdpicsum.dev/photo/category/travel/800/600.webp?n=bmw-2', 'image/webp', 120000, 'bmw-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000203', '00000000-0000-0000-0004-000000000002', 'external', 'https://placeholdpicsum.dev/photo/category/travel/800/600.webp?n=bmw-3', 'image/webp', 120000, 'bmw-3.webp', 2, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Apartment (3 photos) — architecture
    ('00000000-0000-0000-0005-000000000301', '00000000-0000-0000-0004-000000000003', 'external', 'https://placeholdpicsum.dev/photo/category/architecture/800/600.webp?n=apartment-1', 'image/webp', 120000, 'apartment-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000302', '00000000-0000-0000-0004-000000000003', 'external', 'https://placeholdpicsum.dev/photo/category/architecture/800/600.webp?n=apartment-2', 'image/webp', 120000, 'apartment-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000303', '00000000-0000-0000-0004-000000000003', 'external', 'https://placeholdpicsum.dev/photo/category/architecture/800/600.webp?n=apartment-3', 'image/webp', 120000, 'apartment-3.webp', 2, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Winter jacket (2 photos) — people
    ('00000000-0000-0000-0005-000000000401', '00000000-0000-0000-0004-000000000004', 'external', 'https://placeholdpicsum.dev/photo/category/people/800/600.webp?n=jacket-1', 'image/webp', 120000, 'jacket-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000402', '00000000-0000-0000-0004-000000000004', 'external', 'https://placeholdpicsum.dev/photo/category/people/800/600.webp?n=jacket-2', 'image/webp', 120000, 'jacket-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Dining table (2 photos) — architecture
    ('00000000-0000-0000-0005-000000000501', '00000000-0000-0000-0004-000000000005', 'external', 'https://placeholdpicsum.dev/photo/category/architecture/800/600.webp?n=table-1', 'image/webp', 120000, 'table-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000502', '00000000-0000-0000-0004-000000000005', 'external', 'https://placeholdpicsum.dev/photo/category/architecture/800/600.webp?n=table-2', 'image/webp', 120000, 'table-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Mountain bike (2 photos) — travel
    ('00000000-0000-0000-0005-000000000601', '00000000-0000-0000-0004-000000000006', 'external', 'https://placeholdpicsum.dev/photo/category/travel/800/600.webp?n=bike-1', 'image/webp', 120000, 'bike-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000602', '00000000-0000-0000-0004-000000000006', 'external', 'https://placeholdpicsum.dev/photo/category/travel/800/600.webp?n=bike-2', 'image/webp', 120000, 'bike-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Job vacancy (2 photos) — business
    ('00000000-0000-0000-0005-000000000701', '00000000-0000-0000-0004-000000000007', 'external', 'https://placeholdpicsum.dev/photo/category/business/800/600.webp?n=job-1', 'image/webp', 120000, 'job-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000702', '00000000-0000-0000-0004-000000000007', 'external', 'https://placeholdpicsum.dev/photo/category/business/800/600.webp?n=job-2', 'image/webp', 120000, 'job-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    -- Aquarium (2 photos) — animals
    ('00000000-0000-0000-0005-000000000801', '00000000-0000-0000-0004-000000000008', 'external', 'https://placeholdpicsum.dev/photo/category/animals/800/600.webp?n=aquarium-1', 'image/webp', 120000, 'aquarium-1.webp', 0, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1)),
    ('00000000-0000-0000-0005-000000000802', '00000000-0000-0000-0004-000000000008', 'external', 'https://placeholdpicsum.dev/photo/category/animals/800/600.webp?n=aquarium-2', 'image/webp', 120000, 'aquarium-2.webp', 1, (SELECT id FROM users WHERE email = 'test@example.com' LIMIT 1));

-- Set each item's thumbnail to its first photo (sort_order = 0).
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000101' WHERE id = '00000000-0000-0000-0004-000000000001';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000201' WHERE id = '00000000-0000-0000-0004-000000000002';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000301' WHERE id = '00000000-0000-0000-0004-000000000003';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000401' WHERE id = '00000000-0000-0000-0004-000000000004';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000501' WHERE id = '00000000-0000-0000-0004-000000000005';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000601' WHERE id = '00000000-0000-0000-0004-000000000006';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000701' WHERE id = '00000000-0000-0000-0004-000000000007';
UPDATE items SET thumbnail_id = '00000000-0000-0000-0005-000000000801' WHERE id = '00000000-0000-0000-0004-000000000008';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS paid_entitlements;
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS chat_reads;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS refresh_sessions;
DROP TABLE IF EXISTS user_favorites;
DROP TABLE IF EXISTS user_avatars;
DROP TABLE IF EXISTS item_photos;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS cities;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS payment_event_status;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_purpose;
DROP TYPE IF EXISTS payment_provider;
DROP TYPE IF EXISTS item_status;
DROP TYPE IF EXISTS item_condition;

-- +goose StatementEnd
