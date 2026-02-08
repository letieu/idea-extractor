package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/letieu/idea-extractor/config"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cnf, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
		return
	}

	dbPath := cnf.Database.DBName

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Fail to create db file %v", err)
	}

	sqlite_vec.Auto()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var vecVersion string
	err = db.QueryRow("select vec_version()").Scan(&vecVersion)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("vec_version=%s\n", vecVersion)

	sqlBytes, err := os.ReadFile("init.sql")
	if err != nil {
		log.Fatalf("Fail to read sql init file %v", err)
	}

	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		log.Fatalf("Fail to run sql init file %v", err)
	}

	log.Println("DONE")
}
