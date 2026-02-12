package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/letieu/idea-extractor/config"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	cnf, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
		return
	}

	dbUrl := cnf.Database.Url
	token := cnf.Database.Token
	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", dbUrl, token))

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlBytes, err := os.ReadFile("init.sql")
	if err != nil {
		log.Fatalf("Fail to read sql init file %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	for _, stmt := range strings.Split(string(sqlBytes), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			log.Fatalf("SQL failed:\n%s\nERROR: %v", stmt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	log.Println("DONE")
}
