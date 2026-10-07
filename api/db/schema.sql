CREATE TABLE users (
  id CHAR(36) NOT NULL,
  email VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  last_activity_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY users_email_unique (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE wallets (
  user_id CHAR(36) NOT NULL,
  balance_minor BIGINT NOT NULL,
  PRIMARY KEY (user_id),
  CONSTRAINT wallets_user_fk FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE transfers (
  sender_id CHAR(36) NOT NULL,
  id VARCHAR(64) NOT NULL,
  recipient_id CHAR(36) NOT NULL,
  amount_minor BIGINT NOT NULL,
  notes VARCHAR(200) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (sender_id, id),
  KEY transfers_recipient_id (recipient_id),
  CONSTRAINT transfers_sender_fk FOREIGN KEY (sender_id) REFERENCES users (id),
  CONSTRAINT transfers_recipient_fk FOREIGN KEY (recipient_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
