DB_URL=postgres://chat:secret@localhost:25433/chat?sslmode=disable

migrateup:
	migrate -path migrations -database "$(DB_URL)" -verbose up
migratedown:
	migrate -path migrations -database "$(DB_URL)" -verbose down
sqlc:
	sqlc generate
run:
	go run cmd/server/main.go

.PHONY: migrateup migratedown sqlc run