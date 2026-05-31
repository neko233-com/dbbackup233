CREATE TABLE guilds (
  id BIGINT PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE players (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  guild_id BIGINT NULL,
  name VARCHAR(64) NOT NULL,
  nickname VARCHAR(64) NULL,
  level INT NOT NULL,
  gold BIGINT NOT NULL,
  diamond DECIMAL(20,4) NOT NULL,
  win_rate DOUBLE NOT NULL,
  active TINYINT(1) NOT NULL,
  role ENUM('warrior','mage','archer') NOT NULL,
  flags SET('newbie','vip','muted') NULL,
  profile JSON NULL,
  avatar BLOB NULL,
  bio TEXT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_level_gold (level, gold),
  CONSTRAINT fk_players_guild FOREIGN KEY (guild_id) REFERENCES guilds(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE inventory_items (
  player_id BIGINT NOT NULL,
  slot_no INT NOT NULL,
  item_id BIGINT NOT NULL,
  amount INT NOT NULL,
  attrs JSON NULL,
  PRIMARY KEY (player_id, slot_no),
  CONSTRAINT fk_inventory_player FOREIGN KEY (player_id) REFERENCES players(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
