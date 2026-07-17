ALTER TABLE message ADD COLUMN content_tsv tsvector
  GENERATED ALWAYS AS (to_tsvector('english', content)) STORED;

CREATE INDEX idx_message_content_tsv ON message USING GIN (content_tsv);