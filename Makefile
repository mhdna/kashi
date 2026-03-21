postgres:
	docker run --name postgres -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root mercato

dropdb:
	docker exec -it postgres dropdb mercato

migrateup:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/mercato?sslmode=disable" -verbose up

migratedown:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/mercato?sslmode=disable" -verbose down

sqlc:
	sqlc generate
	
test:
	go test -v -cover ./...

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
