-- Seed data for local development. Safe to run more than once.
-- All dummy users share the password: password123

INSERT INTO users (id, email, password_hash, name) VALUES
    ('11111111-1111-1111-1111-111111111111', 'alice@example.com', '$2a$10$.aBbIijYa7jnwQfL/SVABesE2yzYowF.XhFoXlpqQ5O2hyrKyACwC', 'Alice'),
    ('22222222-2222-2222-2222-222222222222', 'bob@example.com',   '$2a$10$.aBbIijYa7jnwQfL/SVABesE2yzYowF.XhFoXlpqQ5O2hyrKyACwC', 'Bob'),
    ('33333333-3333-3333-3333-333333333333', 'carol@example.com', '$2a$10$.aBbIijYa7jnwQfL/SVABesE2yzYowF.XhFoXlpqQ5O2hyrKyACwC', 'Carol')
ON CONFLICT (id) DO NOTHING;

INSERT INTO teams (id, name) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Engineering')
ON CONFLICT (id) DO NOTHING;

INSERT INTO team_members (team_id, user_id, role) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'owner'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-2222-2222-2222-222222222222', 'member'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '33333333-3333-3333-3333-333333333333', 'member')
ON CONFLICT (team_id, user_id) DO NOTHING;

-- Default statuses for each seeded user.
INSERT INTO task_statuses (user_id, name, slug, position, is_default)
SELECT u.id, s.name, s.slug, s.position, true
FROM users u
CROSS JOIN (VALUES
    ('Todo', 'todo', 0),
    ('In Progress', 'in_progress', 1),
    ('Done', 'done', 2)
) AS s(name, slug, position)
WHERE u.id IN (
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    '33333333-3333-3333-3333-333333333333'
)
ON CONFLICT (user_id, slug) DO NOTHING;

-- A couple of sample tasks for Alice.
INSERT INTO tasks (team_id, creator_id, assignee_id, status_id, title, description)
SELECT 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
       '22222222-2222-2222-2222-222222222222', s.id, 'Set up CI pipeline',
       'Configure GitHub Actions for build and test'
FROM task_statuses s
WHERE s.user_id = '11111111-1111-1111-1111-111111111111' AND s.slug = 'todo'
  AND NOT EXISTS (SELECT 1 FROM tasks WHERE title = 'Set up CI pipeline');

INSERT INTO tasks (team_id, creator_id, assignee_id, status_id, title, description)
SELECT 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
       NULL, s.id, 'Write architecture notes',
       'Document the layering and transaction boundaries'
FROM task_statuses s
WHERE s.user_id = '11111111-1111-1111-1111-111111111111' AND s.slug = 'in_progress'
  AND NOT EXISTS (SELECT 1 FROM tasks WHERE title = 'Write architecture notes');
