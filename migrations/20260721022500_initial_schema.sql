-- +goose Up
CREATE TABLE app_user (
    id                UUID PRIMARY KEY,
    email             VARCHAR(255) NOT NULL,
    current_avatar_id UUID         NULL,
    created_at        TIMESTAMPTZ  NOT NULL,
    updated_at        TIMESTAMPTZ  NOT NULL
);
CREATE UNIQUE INDEX app_user_email_unique_idx ON app_user (email);
CREATE INDEX app_user_current_avatar_id_idx ON app_user (current_avatar_id) WHERE current_avatar_id IS NOT NULL;

CREATE TYPE avatar_status AS ENUM (
    'processing',
    'ready',
    'failed',
    'deleting',
    'deleted'
);

CREATE TABLE avatar (
    id                   UUID PRIMARY KEY,
    user_id              UUID          NOT NULL REFERENCES app_user(id) ON DELETE RESTRICT,
    file_name            VARCHAR(255)  NOT NULL,
    mime_type            VARCHAR(255)  NOT NULL,
    size_bytes           BIGINT        NOT NULL,
    width                INTEGER       NULL,
    height               INTEGER       NULL,
    object_key_original  VARCHAR(512)  NOT NULL,
    object_key_thumb_100 VARCHAR(512)  NULL,
    object_key_thumb_300 VARCHAR(512)  NULL,
    status               avatar_status NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL,
    updated_at           TIMESTAMPTZ   NOT NULL,
    deleted_at           TIMESTAMPTZ   NULL
);
ALTER TABLE app_user
    ADD CONSTRAINT app_user_current_avatar_id_fk
    FOREIGN KEY (current_avatar_id) REFERENCES avatar(id) ON DELETE SET NULL;

CREATE INDEX avatar_user_id_active_idx ON avatar (user_id) WHERE deleted_at IS NULL;
CREATE INDEX avatar_status_idx ON avatar (status);

-- +goose Down
DROP TABLE IF EXISTS avatar;
DROP TABLE IF EXISTS app_user;
DROP TYPE IF EXISTS avatar_status;
