//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres",
		"host=172.20.40.2 port=5432 user=jevon_user password=jevon_user dbname=jevon_crm sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Смотрим текущее состояние
	rows, _ := db.Query(`SELECT version, dirty FROM schema_migrations`)
	defer rows.Close()
	fmt.Println("Current schema_migrations:")
	for rows.Next() {
		var version int
		var dirty bool
		rows.Scan(&version, &dirty)
		fmt.Printf("  version=%d dirty=%v\n", version, dirty)
	}

	// Удаляем все записи и ставим version=16 clean
	db.Exec(`DELETE FROM schema_migrations`)
	db.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (16, false)`)

	fmt.Println("✅ schema_migrations reset to version 16 (clean)")
}