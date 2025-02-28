package internal

import (
	"context"
	"math"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Route struct {
	ID           string      `bson:"_id" json:"id"`
	Distance     int         `bson:"distance" json:"distance"`
	Directions   []Direction `bson:"directions" json:"directions"`
	FreightPrice float64     `bson:"freight_price" json:"freight_price"`
}

type Direction struct {
	Lat float64 `bson:"lat" json:"lat"`
	Lng float64 `bson:"lng" json:"lng"`
}

type RouteService struct {
	mongo          *mongo.Client
	freightService *FreightService
}

func NewRoute(id string, distance int, directions []Direction) *Route {
	return &Route{
		ID:         id,
		Distance:   distance,
		Directions: directions,
	}
}

func NewRouteService(mongo *mongo.Client, freightService *FreightService) *RouteService {
	return &RouteService{
		mongo,
		freightService,
	}
}

type FreightService struct{}

func NewFreightService() *FreightService {
	return &FreightService{}
}

func (fs *FreightService) Calculate(distance int) float64 {
	return math.Floor((float64(distance)*0.15+0.3)*100) / 100
}

func (rs *RouteService) CreateRoute(route *Route) (*Route, error) {
	route.FreightPrice = rs.freightService.Calculate(route.Distance)
	update := bson.M{
		"$set": bson.M{
			"distance":     route.Distance,
			"directions":   route.Directions,
			"freightPrice": route.FreightPrice,
		},
	}
	filter := bson.M{"_id": route.ID}
	opts := options.Update().SetUpsert(true)
	_, err := rs.mongo.Database("routes").Collection("routes").UpdateOne(
		context.TODO(), filter, update, opts,
	)
	if err != nil {
		return &Route{}, err
	}
	return route, nil
}

func (rs *RouteService) GetRoute(id string) (Route, error) {
	var route Route
	filter := bson.M{"_id": id}
	err := rs.mongo.Database("routes").Collection("routes").FindOne(context.TODO(), filter).Decode(&route)
	if err != nil {
		return Route{}, err
	}
	return route, nil
}
