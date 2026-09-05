CREATE TABLE order_sagas
(
    id         SERIAL PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL UNIQUE,
    status     VARCHAR(50)  NOT NULL,
    items      JSONB        NOT NULL DEFAULT '[]'::jsonb,
    cancelled  BOOLEAN      NOT NULL DEFAULT FALSE,
    user_id    INT          NOT NULL,
    order_id   INT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_sagas_request_id ON order_sagas (request_id);
