CREATE TABLE tickets (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    subject varchar(255) NOT NULL,
    priority varchar(32) NOT NULL DEFAULT 'medium',
    status varchar(32) NOT NULL DEFAULT 'open',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ticket_messages (
    id uuid PRIMARY KEY,
    ticket_id uuid NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sender_type varchar(32) NOT NULL,
    sender_id uuid NOT NULL,
    message text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_user_id ON tickets(user_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
