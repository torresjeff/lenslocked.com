package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "lenslocked_dev"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		panic(err)
	} else {
		fmt.Println("Ping successful")
	}

	// Insert a new record and get the resulting ID (id is the primary key of the table which auto increments (using SERIAL constraint in Postgres))
	row := db.QueryRow(`
		INSERT INTO users(name, email)
		VALUES ($1, $2) RETURNING id`,
		"Jeff Torres", "torres.jeff@hotmail.com")
	var id int
	err = row.Scan(&id)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("ID of created user:", id)
	}

	// Query a single row
	var name, email string
	row = db.QueryRow(`
		SELECT id, name, email FROM users WHERE id=$1
	`, 1)
	err = row.Scan(&id, &name, &email)
	if err != nil {
		panic(err)
	}
	fmt.Println("ID:", id, "Name:", name, "Email:", email)

	// Query multiple records
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		rows.Scan(&id, &name, &email)
		fmt.Println("ID:", id, "Name:", name, "Email:", email)
	}
}
