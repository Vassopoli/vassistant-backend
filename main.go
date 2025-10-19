package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"vassistant-backend/financial"
	"vassistant-backend/messages"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func init() {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create DynamoDB client
	dynamoDbClient := dynamodb.NewFromConfig(cfg)
	messages.DynamoDbClient = dynamoDbClient
	financial.DynamoDbClient = dynamoDbClient
}

// ErrorResponse struct for JSON error messages
type ErrorResponse struct {
	Error string `json:"error"`
}

// createErrorResponse is a helper function to generate a JSON error response
func createErrorResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(ErrorResponse{Error: message})
	if err != nil {
		// This should not happen, but if it does, log it and return a generic error
		log.Printf("Failed to marshal error response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"error":"Internal server error"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

func rootHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("Request path:", request.Path)
	log.Println("Request HTTP method:", request.HTTPMethod)

	if request.Path == "/VassistantBackendProxy/messages" {
		switch request.HTTPMethod {
		case "POST":
			return messages.PostMessageHandler(request)
		case "GET":
			return messages.GetMessageHandler(request)
		default:
			return createErrorResponse(405, "Method Not Allowed")
		}
	}

	if strings.HasPrefix(request.Path, "/VassistantBackendProxy/financial/groups") {
		parts := strings.Split(request.Path, "/")
		if len(parts) == 4 && parts[3] == "groups" {
			// Path is /financial/groups
			if request.HTTPMethod == "GET" {
				return financial.GetGroupsHandler(request)
			} else {
				return createErrorResponse(405, "Method Not Allowed")
			}
		} else if len(parts) == 6 && parts[3] == "groups" && parts[5] == "expenses" {
			// Path is /financial/groups/{groupId}/expenses
			if request.HTTPMethod == "GET" {
				// The groupId is the 4th part of the path
				request.PathParameters = map[string]string{"groupId": parts[4]}
				return financial.GetGroupExpensesHandler(request)
			} else {
				return createErrorResponse(405, "Method Not Allowed")
			}
		}
	}

	return createErrorResponse(404, "Not Found")
}

func main() {
	lambda.Start(rootHandler)
}
