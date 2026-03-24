postgres:
	docker run --name postgres -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root kashi

dropdb:
	docker exec -it postgres dropdb kashi

migrateup:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose up

migratedown:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose down

sqlc:
	sqlc generate
	
test:
	go test -v -cover ./...

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
