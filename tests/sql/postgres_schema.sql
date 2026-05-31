CREATE TABLE guilds (
  id BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE players (
  id BIGSERIAL PRIMARY KEY,
  guild_id BIGINT REFERENCES guilds(id),
  name TEXT NOT NULL,
  level INTEGER NOT NULL,
  gold BIGINT NOT NULL,
  diamond NUMERIC(20,4) NOT NULL,
  win_rate DOUBLE PRECISION NOT NULL,
  active BOOLEAN NOT NULL,
  profile JSONB,
  bio TEXT,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_players_level_gold ON players(level, gold);
