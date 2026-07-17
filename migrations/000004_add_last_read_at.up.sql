ALTER TABLE conversation_participant
ADD COLUMN last_read_at TIMESTAMPTZ NOT NULL DEFAULT now();