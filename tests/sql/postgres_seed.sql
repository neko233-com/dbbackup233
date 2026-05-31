INSERT INTO guilds(id, name, created_at) VALUES
  (1, 'Alpha guild', '2026-05-31T10:00:00Z'),
  (2, 'Beta', '2026-05-31T10:01:00Z');

INSERT INTO players(guild_id, name, level, gold, diamond, win_rate, active, profile, bio, created_at) VALUES
  (1, 'alice', 10, 100, 12.3400, 0.88, true, '{"server":"s1"}', 'hello', '2026-05-31T11:00:00Z'),
  (1, 'bob', 20, 250, 99.9900, 0.42, false, '{"server":"s1","muted":true}', NULL, '2026-05-31T11:01:00Z');
