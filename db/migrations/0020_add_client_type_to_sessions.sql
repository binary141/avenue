-- Records which kind of client a session was created by. The allowed values
-- live in session_types, and sessions.session_type_id references them.
-- Existing sessions were all created by the web client, so they get 'web'.
BEGIN;

CREATE TABLE session_types (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO session_types (name) VALUES ('web'), ('mobile');

ALTER TABLE sessions ADD COLUMN session_type_id BIGINT REFERENCES session_types (id);

UPDATE sessions
SET session_type_id = (SELECT id FROM session_types WHERE name = 'web');

ALTER TABLE sessions ALTER COLUMN session_type_id SET NOT NULL;

COMMIT;
