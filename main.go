package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	// 	// urlExample := "postgres://username:password@localhost:5432/database_name"
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://amza:1@localhost:5432/natsdlqexample")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to database")

	_, err = conn.Exec(context.Background(), "INSERT INTO entries (timestamp) VALUES ($1)", time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to insert")
		os.Exit(1)
	}

	fmt.Println("Success in the insertion!")

	//steps to do database entry
	//conn.exec(context.Background), ""

	defer conn.Close(context.Background())

	for _ = range time.Tick(time.Second) {
		insert(ctx, conn)
		fmt.Println("Insert success!")
	}

}

//context provides a way to:
//carry deadlines
//cancellation signals
//request-scoped values
//for example, helping to pass small variable between functions without relying on global variables

func insert(ctx context.Context, conn *pgx.Conn) {
	ts := time.Now()
	conn.Exec(ctx, "INSERT INTO entries (timestamp) VALUES ($1)", ts)
	fmt.Printf("Inserted entry with timestamp: %v\n", ts)

}
