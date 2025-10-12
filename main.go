package main

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Message struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Content  string `json:"content"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)
	authorizer := request.RequestContext.Authorizer
	sub, _ := authorizer["sub"].(string)
	username, _ := authorizer["username"].(string)

	message := Message{
		Sub:      sub,
		Username: username,
		Content:  "mock",
	}

	responseBody, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling json:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal server error",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(responseBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}