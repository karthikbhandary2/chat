-- name: CreateConversation :one
INSERT INTO conversation (type)
VALUES ($1)
RETURNING *;

-- name: AddParticipant :one
INSERT INTO conversation_participant (conversation_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: CreateMessage :one
INSERT INTO message (conversation_id, sender_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetConversationMessages :many
SELECT * FROM message
WHERE conversation_id = $1
  AND created_at < $2
ORDER BY created_at DESC
LIMIT $3;

-- name: IsParticipant :one
SELECT EXISTS (
    SELECT 1
    FROM conversation_participant
    WHERE conversation_id = $1
        AND user_id = $2
) AS is_participant;

-- name: GetConversationParticipants :many
SELECT user_id FROM conversation_participant
WHERE conversation_id = $1;

-- name: GetUnreadCount :one
SELECT COUNT(*) FROM message m
JOIN conversation_participant cp ON cp.conversation_id = m.conversation_id
WHERE m.conversation_id = $1
  AND cp.user_id = $2
  AND m.created_at > cp.last_read_at;

-- name: MarkAsRead :exec
UPDATE conversation_participant
SET last_read_at = now()
WHERE conversation_id = $1 AND user_id = $2;

-- name: SearchMessages :many
SELECT m.* FROM message m
JOIN conversation_participant cp ON cp.conversation_id = m.conversation_id
WHERE cp.user_id = $1
  AND m.content_tsv @@ plainto_tsquery('english', $2)
ORDER BY m.created_at DESC
LIMIT $3;

-- name: ListUserConversations :many
SELECT c.* FROM conversation c
JOIN conversation_participant cp ON cp.conversation_id = c.id
WHERE cp.user_id = $1
ORDER BY c.created_at DESC;