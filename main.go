package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/mhdna/kashi/api"
	db "github.com/mhdna/kashi/db/sqlc"
)

const (
	address  = "0.0.0.0:8080"
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5433/kashi?sslmode=disable"
)

func main() {

	var err error
	conn, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(address)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
