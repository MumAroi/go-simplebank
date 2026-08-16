DB_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
PROTO_FILES := $(wildcard proto/*.proto)

createdb:
	docker exec -it simplebank-postgres createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it simplebank-postgres dropdb simple_bank

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mock:
	mockgen --package mockdb -destination db/mock/store.go github.com/MumAroi/go-simplebank/db/sqlc Store

proto:
	nu -c "rm  -f  pb/*.go"
	protoc	--proto_path=proto --go_out=pb --go_opt=paths=source_relative \
	--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	$(PROTO_FILES)

.PHONY: createdb dropdb migrateup migratedown sqlc test server mock proto
