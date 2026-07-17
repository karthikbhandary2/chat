DROP INDEX IF EXISTS idx_message_content_tsv;
ALTER TABLE message DROP COLUMN content_tsv;