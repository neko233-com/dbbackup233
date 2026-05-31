INSERT INTO guilds(id, name, created_at) VALUES
  (1, 'Alpha 公会', '2026-05-31 10:00:00.123456'),
  (2, 'Beta', '2026-05-31 10:01:00.000001');

INSERT INTO players(guild_id, name, nickname, level, gold, diamond, win_rate, active, role, flags, profile, avatar, bio, created_at) VALUES
  (1, 'alice', '队长', 10, 100, 12.3400, 0.88, 1, 'warrior', 'newbie,vip', JSON_OBJECT('server','s1','skins',JSON_ARRAY('red','blue')), X'CAFEBABE', 'hello\nworld', '2026-05-31 11:00:00.654321'),
  (1, 'bob', NULL, 20, 250, 99.9900, 0.42, 0, 'mage', NULL, JSON_OBJECT('server','s1','muted',true), X'00FF', NULL, '2026-05-31 11:01:00.000001'),
  (2, 'emoji-😀', 'utf8mb4', 30, 999999999, 123456.7890, 1.0, 1, 'archer', 'vip', JSON_OBJECT('note','emoji 😀'), NULL, 'unicode ok', '2026-05-31 11:02:00.000002');

INSERT INTO inventory_items(player_id, slot_no, item_id, amount, attrs) VALUES
  (1, 1, 10001, 5, JSON_OBJECT('quality','rare')),
  (1, 2, 10002, 1, NULL),
  (2, 1, 20001, 10, JSON_OBJECT('expire_at','2026-06-30T00:00:00Z')),
  (3, 1, 30001, 99, JSON_OBJECT('stackable',true));
