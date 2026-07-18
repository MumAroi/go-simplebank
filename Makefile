createdb:
	podman exec -it simplebank-postgres createdb --username=root --owner=root simple_bank

dropdb:
	podman exec -it simplebank-postgres dropdb simple_bank

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" down

sqlc:
	sqlc generate

.PHONY: createdb dropdb migrateup migratedown
