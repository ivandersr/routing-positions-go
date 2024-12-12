package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventHub struct {
	routeService     *RouteService
	mongoClient      *mongo.Client
	chDriverMoved    chan *DriverMovedEvent
	freightWriter    *kafka.Writer
	simulationWriter *kafka.Writer
}

func NewEventHub(
	routeService *RouteService,
	mongoClient *mongo.Client,
	chDriverMoved chan *DriverMovedEvent,
	freightWriter *kafka.Writer,
	simulationWriter *kafka.Writer,
) *EventHub {
	return &EventHub{
		routeService,
		mongoClient,
		chDriverMoved,
		freightWriter,
		simulationWriter,
	}
}

func (eh *EventHub) HandleEvent(msg []byte) error {
	var baseEvent struct {
		EventName string `json:"event"`
	}
	err := json.Unmarshal(msg, &baseEvent)
	if err != nil {
		return fmt.Errorf("error unmarshalling event: %w", err)
	}
	switch baseEvent.EventName {
	case "RouteCreated":
		var event RouteCreatedEvent
		err = json.Unmarshal(msg, &event)
		if err != nil {
			return fmt.Errorf("error unmarshalling event: %w", err)
		}
		return eh.HandleRouteCreated(event)

	case "DeliveryStarted":
		var event DeliveryStartedEvent
		err = json.Unmarshal(msg, &event)
		if err != nil {
			return fmt.Errorf("error unmarshalling event: %w", err)
		}
	default:
		return errors.New("unknown event")
	}
	return nil
}

func (eh *EventHub) HandleRouteCreated(event RouteCreatedEvent) error {
	freightCalculatedEvent, err := RouteCreatedHandler(&event, eh.routeService)
	if err != nil {
		return err
	}
	value, err := json.Marshal(freightCalculatedEvent)
	if err != nil {
		return fmt.Errorf("error unmarshalling event: %w", err)
	}
	err = eh.freightWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(freightCalculatedEvent.RouteID),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("error writing to topic: %w", err)
	}
	return nil
}

func (eh *EventHub) HandleDeliveryStarted(event DeliveryStartedEvent) error {
	err := DeliveryStartedHandler(&event, eh.routeService, eh.chDriverMoved)
	if err != nil {
		return err
	}
	go eh.sendDirections() // parallel processing to avoid blocking execution
	return nil
}

func (eh *EventHub) sendDirections() {
	for {
		select {
		case movedEvent := <-eh.chDriverMoved:
			value, err := json.Marshal(movedEvent)
			if err != nil {
				return
			}
			err = eh.simulationWriter.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte(movedEvent.RouteID),
				Value: value,
			})
			if err != nil {
				return
			}
		case <-time.After(500 * time.Millisecond): // close thread to avoid leaks
			return
		}
	}
}
