-- +goose Up

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254),
    display_name  TEXT        NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 60),
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- В cookie — случайный токен, здесь хранится только его SHA-256.
CREATE TABLE user_sessions (
    token_hash BYTEA       PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX user_sessions_user_idx ON user_sessions (user_id);

CREATE TABLE workspaces (
    id          BIGSERIAL PRIMARY KEY,
    owner_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    description TEXT        NOT NULL DEFAULT '' CHECK (char_length(description) <= 500),
    color       TEXT        NOT NULL DEFAULT 'violet',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX workspaces_owner_idx ON workspaces (owner_id);

CREATE TABLE folders (
    id           BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT      NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    description  TEXT        NOT NULL DEFAULT '' CHECK (char_length(description) <= 500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX folders_workspace_idx ON folders (workspace_id);

CREATE TABLE cards (
    id         BIGSERIAL PRIMARY KEY,
    folder_id  BIGINT      NOT NULL REFERENCES folders (id) ON DELETE CASCADE,
    ru_word    TEXT        NOT NULL CHECK (char_length(ru_word) BETWEEN 1 AND 200),
    ru_example TEXT        NOT NULL DEFAULT '' CHECK (char_length(ru_example) <= 1000),
    en_word    TEXT        NOT NULL CHECK (char_length(en_word) BETWEEN 1 AND 200),
    en_example TEXT        NOT NULL DEFAULT '' CHECK (char_length(en_example) <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX cards_folder_idx ON cards (folder_id);

CREATE TABLE share_links (
    token        TEXT PRIMARY KEY,
    owner_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    workspace_id BIGINT REFERENCES workspaces (id) ON DELETE CASCADE,
    folder_id    BIGINT REFERENCES folders (id) ON DELETE CASCADE,
    card_id      BIGINT REFERENCES cards (id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (num_nonnulls(workspace_id, folder_id, card_id) = 1)
);
CREATE UNIQUE INDEX share_links_workspace_key ON share_links (workspace_id) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX share_links_folder_key ON share_links (folder_id) WHERE folder_id IS NOT NULL;
CREATE UNIQUE INDEX share_links_card_key ON share_links (card_id) WHERE card_id IS NOT NULL;
CREATE INDEX share_links_owner_idx ON share_links (owner_id);

CREATE TABLE study_sessions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mode         TEXT        NOT NULL CHECK (mode IN ('learn', 'dictation', 'match')),
    workspace_id BIGINT REFERENCES workspaces (id) ON DELETE SET NULL,
    folder_id    BIGINT REFERENCES folders (id) ON DELETE SET NULL,
    title        TEXT        NOT NULL DEFAULT '' CHECK (char_length(title) <= 200),
    total_cards  INT         NOT NULL CHECK (total_cards >= 0),
    mistakes     INT         NOT NULL CHECK (mistakes >= 0),
    rounds       INT         NOT NULL CHECK (rounds >= 0),
    duration_ms  BIGINT      NOT NULL CHECK (duration_ms >= 0),
    completed    BOOLEAN     NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX study_sessions_user_created_idx ON study_sessions (user_id, created_at DESC);

-- +goose Down
DROP TABLE study_sessions;
DROP TABLE share_links;
DROP TABLE cards;
DROP TABLE folders;
DROP TABLE workspaces;
DROP TABLE user_sessions;
DROP TABLE users;
