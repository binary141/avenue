-- Auth (identity, passwords, sessions) moves to the keyring service. This
-- repurposes the local users table into a thin quota mirror keyed off
-- keyring's user, and drops the tables keyring now owns. Local user/session
-- data is dev-disposable, so this truncates rather than migrating rows.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

TRUNCATE TABLE users CASCADE;

ALTER TABLE users
    DROP COLUMN email,
    DROP COLUMN first_name,
    DROP COLUMN last_name,
    DROP COLUMN password,
    DROP COLUMN can_login,
    DROP COLUMN is_admin;

ALTER TABLE users
    ADD COLUMN keyring_user_id BIGINT NOT NULL,
    ADD COLUMN keyring_uuid    UUID   NOT NULL;

CREATE UNIQUE INDEX idx_users_keyring_user_id ON users (keyring_user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_keyring_uuid ON users (keyring_uuid) WHERE deleted_at IS NULL;

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS sessions;
