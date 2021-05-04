package main

import (
	"context"
	cryptoRand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	mathRand "math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Unicorn struct {
	Name   string `json:"Name"`
	Color  string `json:"Color"`
	Gender string `json:"Gender"`
}

type Request struct {
	PickupLocation struct {
		Latitude  float64 `json:"Latitude"`
		Longitude float64 `json:"Longitude"`
	} `json:"PickupLocation"`
}

type Headers struct {
	AccessControlAllowOrigin string `json:"Access-Control-Allow-Origin"`
}

type Response struct {
	StatusCode      int          `json:"statusCode"`
	Body            ResponseBody `json:"body"`
	Headers         Headers      `json:"headers"`
	IsBase64Encoded bool         `json:"isBase64Encoded"`
}

type ResponseBody struct {
	RideId      string `json:"RideId"`
	Unicorn     `json:"Unicorn"`
	UnicornName string `json:"UnicornName"`
	Eta         string `json:"Eta"`
	Rider       string `json:"Rider"`
}

type DBItem struct {
	RideId      string
	User        string
	Unicorn     Unicorn
	UnicornName string
	RequestTime string
}

var fleet = [3]Unicorn{
	{"Bucephalus", "Golgen", "Male"},
	{"Bucephalus", "Golgen", "Male"},
	{"Rocinante", "Yellow", "Female"},
}

func HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	rideId, _ := newUUID()

	fmt.Printf("Received event (%d): %+v", &rideId, event)
	fmt.Println()

	requestBody := Request{}
	json.Unmarshal([]byte(event.Body), &requestBody)

	latitude := requestBody.PickupLocation.Latitude
	longitude := requestBody.PickupLocation.Longitude
	unicorn := findUnicorn(latitude, longitude)
	username := event.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)

	recordRide(DBItem{
		rideId,
		username,
		unicorn,
		unicorn.Name,
		time.Now().Format(time.RFC3339),
	})

	responseBody, _ := json.Marshal(ResponseBody{rideId, unicorn, unicorn.Name, "30 seconds", username})
	responseBodyRaw := json.RawMessage(responseBody)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBodyRaw),
		Headers:    map[string]string{"Access-Control-Allow-Origin": "*"},
	}, nil
}

func recordRide(ride DBItem) error {
	// create an aws session
	mySession := session.Must(session.NewSession())

	// create a dynamodb instance
	ddb := dynamodb.New(mySession)

	// marshal the movie struct into an aws attribute value
	rideAVMap, err := dynamodbattribute.MarshalMap(ride)
	if err != nil {
		panic("Cannot marshal ride into AttributeValue map")
	}

	// create the api params
	params := &dynamodb.PutItemInput{
		TableName: aws.String("Rides"),
		Item:      rideAVMap,
	}

	// put the item
	resp, err := ddb.PutItem(params)
	if err != nil {
		panic(fmt.Sprintf("DynamoDB ERROR: %v\n", err.Error()))
	}

	// print the response data
	fmt.Printf("DynamoDB Success: %s\n", resp)

	return nil
}

func findUnicorn(latitude float64, longitude float64) Unicorn {
	fmt.Printf("Finding unicorn for %f, %f", latitude, longitude)
	fmt.Println()

	unicornId := mathRand.Intn(len(fleet))

	return fleet[unicornId]
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(cryptoRand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func main() {
	lambda.Start(HandleRequest)
}
