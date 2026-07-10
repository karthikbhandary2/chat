CREATE TABLE
    IF NOT EXISTS conversation (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        type TEXT NOT NULL CHECK (type IN ('direct', 'group')),
        created_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

CREATE TABLE
    IF NOT EXISTS conversation_participant (
        conversation_id UUID NOT NULL REFERENCES conversation (id) ON DELETE CASCADE,
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        joined_at TIMESTAMPTZ NOT NULL DEFAULT now (),
        PRIMARY KEY (conversation_id, user_id)
    );

CREATE TABLE
    IF NOT EXISTS message (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        conversation_id UUID NOT NULL REFERENCES conversation (id) ON DELETE CASCADE,
        sender_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        content TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

CREATE INDEX idx_message_conversation_created_at ON message (conversation_id, created_at);