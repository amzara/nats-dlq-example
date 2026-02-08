package main

import (
	"context"
	"fmt"
	"os"
	_ "os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	//format for db conn string is postgresql://user:password@host:port/database
	dbConn := "postgresql://amza:1@127.0.0.1:5432/natsdlqexample"

	actualDBConn, err := pgx.Connect(context.Background(), dbConn)

	if err != nil {
		fmt.Println(err) // maybe add a ping to the db, so if the db conn fail can alert telegram.
	}

	defer actualDBConn.Close(context.Background())

	if err := actualDBConn.Ping(context.Background()); err != nil {
		fmt.Println("DB ping failed:", err)
		os.Exit(1)
	}

	fmt.Println("Db conn success")

	time.Sleep(5 * time.Second)

	opts := &server.Options{
		Port:      4222,
		JetStream: true,
		StoreDir:  "/tmp/nats-dlq",
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		panic(err)
	}
	go ns.Start()

	// Wait for server to be ready
	if !ns.ReadyForConnections(10 * time.Second) {
		panic("server didn't start in time")
	}
	fmt.Println("Server ready!")

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	// 3. Create JetStream
	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	js.DeleteStream(context.Background(), "inputstream")
	//clear previous stream properly on reboots.. tbh no need, persistence is always good but need to find ways to handle

	ctx := context.Background()

	// 4. Create stream
	s, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "inputstream",
		Subjects:  []string{"entries.*"},
		Storage:   jetstream.MemoryStorage,   //but for my case so i do not want any form of retention out of memory
		Retention: jetstream.WorkQueuePolicy, //handles retention (clears the item on the stra)
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Stream 'inputstream' created")

	c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:    "Entries",
		AckPolicy:  jetstream.AckExplicitPolicy,
		MaxDeliver: 3,
		BackOff: []time.Duration{
			60 * time.Second,
			60 * time.Second,
			300 * time.Second,
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Consumer 'Entries' created")

	// 6. Start consuming in background
	cons, err := c.Consume(func(msg jetstream.Msg) {
		fmt.Printf("✓ Received: %s\n", string(msg.Data()))
		err := insert(context.Background(), actualDBConn)
		if err != nil {
			fmt.Printf("Insert not success")
			msg.Nak()
			return
		}

		msg.Ack()
	})
	if err != nil {
		panic(err)
	}
	defer cons.Stop()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			info, err := c.Info(ctx)
			if err != nil {
				fmt.Printf("[Status] Error getting consumer info: %v\n", err)
				continue
			}
			fmt.Printf("[Status %s] Pending: %d | In-flight: %d | Redelivering: %d\n",
				time.Now().Format("15:04:05"),
				info.NumPending,
				info.NumAckPending,
			)
		}
	}()

	fmt.Println("Consumer started, ready to receive messages...")

	// 7. Publish messages (consumer will receive them in real-time)
	fmt.Println("Publishing 100 messages...")
	for i := 0; i < 100; i++ {
		_, err := js.Publish(ctx, "entries.new", []byte("hello message "+strconv.Itoa(i)))
		fmt.Println("Published message $d", i)
		if err != nil {
			fmt.Printf("✗ Publish error: %v\n", err)
			continue
		}
		// Small delay so you can see messages flowing in
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println("Done publishing all messages!")

	info, _ := s.Info(ctx)
	fmt.Printf("Unprocessed messages (likely failed): %d\n", info.State.Msgs)
	// Keep running to allow consumer to process all messages

	time.Sleep(999 * time.Second)
}

//context provides a way to:
//carry deadlines
//cancellation signals
//request-scoped values
//for example, helping to pass small variable between functions without relying on global variables

func insert(ctx context.Context, conn *pgx.Conn) error {
	ts := time.Now()
	_, err := conn.Exec(ctx, "INSERT INTO entries (timestamp) VALUES ($1)", ts)
	if err != nil {
		fmt.Println("Error inserting into DB %s", err)
		return err
	}
	fmt.Printf("Inserted entry: %s\n", ts)
	return nil
}
