-- +goose Up
-- +goose StatementBegin

-- Seed demo content for demo users:
-- - Demo listings owned by demo1, demo2, demo3
-- - Demo favorites by demo1 (pointing to existing seed items)
-- - Demo conversation between demo1 and demo2 with chat messages

-- Demo listings: 4 items across different categories/cities
-- Using UUID prefix 000a for items to keep distinct from existing rows
INSERT INTO items (id, seller_id, category_id, city_id, slug, title, description, price_amount, price_currency, condition, status, created_at, updated_at)
VALUES
    -- Demo1's listings
    ('00000000-0000-0000-000a-000000000001',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0003-000000000001',
     '00000000-0000-0000-0001-000000000001',
     'demo-listing-1-laptop',
     'MacBook Pro 16" 2021',
     'Excellent condition, M1 Max chip, 32GB RAM, 512GB SSD. Minimal use, in original packaging.',
     180000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    ('00000000-0000-0000-000a-000000000002',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0003-000000000004',
     '00000000-0000-0000-0001-000000000002',
     'demo-listing-2-shoes',
     'Running shoes Nike Air Max',
     'Brand new, size 42, never worn, straight from the box.',
     8500,
     'USD',
     'new',
     'published',
     NOW(),
     NOW()),
    -- Demo2's listing
    ('00000000-0000-0000-000a-000000000003',
     '00000000-0000-0000-0009-000000000002',
     '00000000-0000-0000-0003-000000000005',
     '00000000-0000-0000-0001-000000000003',
     'demo-listing-3-bookshelf',
     'Wooden bookshelf 5-tier',
     'Modern design, walnut finish, sturdy and spacious. Height 180cm.',
     25000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW()),
    -- Demo3's listing
    ('00000000-0000-0000-000a-000000000004',
     '00000000-0000-0000-0009-000000000003',
     '00000000-0000-0000-0003-000000000006',
     '00000000-0000-0000-0001-000000000001',
     'demo-listing-4-guitar',
     'Acoustic guitar Yamaha',
     'Well-maintained, full size, includes soft case and strap.',
     42000,
     'USD',
     'used',
     'published',
     NOW(),
     NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo favorites: demo1 favorites some of the existing seed items
-- Using UUID prefix 000b for favorites
INSERT INTO user_favorites (id, user_id, item_id, created_at)
VALUES
    ('00000000-0000-0000-000b-000000000001',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0004-000000000001',
     NOW()),
    ('00000000-0000-0000-000b-000000000002',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0004-000000000004',
     NOW()),
    ('00000000-0000-0000-000b-000000000003',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0004-000000000005',
     NOW())
ON CONFLICT (user_id, item_id) DO NOTHING;

-- Demo conversation between demo1 and demo2
-- CRITICAL: user_a must be the numerically-smaller UUID
-- demo1 (00...0009-...0001) < demo2 (00...0009-...0002), so user_a = demo1, user_b = demo2
-- Using UUID prefix 000c for conversations
INSERT INTO conversations (id, user_a, user_b, created_at, last_message_at)
VALUES
    ('00000000-0000-0000-000c-000000000001',
     '00000000-0000-0000-0009-000000000001',
     '00000000-0000-0000-0009-000000000002',
     NOW(),
     NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo chat messages in the conversation
-- Using UUID prefix 000d for messages
-- Alternating between demo1 and demo2
INSERT INTO chat_messages (id, conversation_id, sender_id, body, created_at)
VALUES
    ('00000000-0000-0000-000d-000000000001',
     '00000000-0000-0000-000c-000000000001',
     '00000000-0000-0000-0009-000000000001',
     'Hi, is this still available?',
     NOW() - INTERVAL '2 hours'),
    ('00000000-0000-0000-000d-000000000002',
     '00000000-0000-0000-000c-000000000001',
     '00000000-0000-0000-0009-000000000002',
     'Yes, it is! I just listed it yesterday.',
     NOW() - INTERVAL '1 hour 45 minutes'),
    ('00000000-0000-0000-000d-000000000003',
     '00000000-0000-0000-000c-000000000001',
     '00000000-0000-0000-0009-000000000001',
     'Great! Can we arrange a time to meet?',
     NOW() - INTERVAL '30 minutes')
ON CONFLICT (id) DO NOTHING;

-- Update conversation last_message_at to the timestamp of the newest message
UPDATE conversations
SET last_message_at = (
    SELECT MAX(created_at) FROM chat_messages
    WHERE conversation_id = '00000000-0000-0000-000c-000000000001'
)
WHERE id = '00000000-0000-0000-000c-000000000001';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Delete demo content in reverse FK dependency order

-- Delete chat messages (prefix 000d)
DELETE FROM chat_messages
WHERE id LIKE '00000000-0000-0000-000d-%';

-- Delete conversations (prefix 000c)
DELETE FROM conversations
WHERE id LIKE '00000000-0000-0000-000c-%';

-- Delete user favorites (prefix 000b)
DELETE FROM user_favorites
WHERE id LIKE '00000000-0000-0000-000b-%';

-- Delete demo items (prefix 000a)
DELETE FROM items
WHERE id LIKE '00000000-0000-0000-000a-%';

-- +goose StatementEnd
