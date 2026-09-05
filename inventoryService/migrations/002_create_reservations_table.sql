CREATE TABLE IF NOT EXISTS reservations (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL,
    order_id INTEGER,
    item_id INTEGER NOT NULL REFERENCES inventory(id),
    quantity INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reservations_request_id ON reservations(request_id);
CREATE INDEX idx_reservations_order_id ON reservations(order_id);
CREATE INDEX idx_reservations_status ON reservations(status);
