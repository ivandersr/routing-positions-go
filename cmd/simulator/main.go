package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ivandersr/routing-positions-go/internal"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoStr := "mongodb://admin:admin@localhost:27018/routes?authSource=admin"
	mongoConn, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoStr))
	if err != nil {
		panic(err)
	}

	freightService := internal.NewFreightService()
	routeService := internal.NewRouteService(mongoConn, freightService)
	chDriverMoved := make(chan *internal.DriverMovedEvent)
	freightWriter := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "freight",
		Balancer: &kafka.LeastBytes{},
	}
	simulatorWriter := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "simulator",
		Balancer: &kafka.LeastBytes{},
	}

	hub := internal.NewEventHub(
		routeService,
		mongoConn,
		chDriverMoved,
		freightWriter,
		simulatorWriter,
	)

	routeReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "route",
		GroupID: "simulator",
	})

	fmt.Println("Starting simulator...")
	for {
		m, err := routeReader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}
		go func(msg []byte) {
			err = hub.HandleEvent(m.Value)
			if err != nil {
				log.Printf("error: %v", err)
			}
		}(m.Value)
	}
}
