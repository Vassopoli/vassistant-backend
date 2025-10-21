package main

import (
	"context"
	"log"
	"vassistant-backend/api"
	"vassistant-backend/financial"
	"vassistant-backend/messages"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var router *api.Router

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

	// Initialize the router
	router = api.NewRouter()
	router.AddRoute("POST", "/VassistantBackendProxy/messages", messages.PostMessageHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/messages", messages.GetMessageHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/groups", financial.GetGroupsHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/groups/(?P<groupId>[^/]+)", financial.GetGroupHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/groups/(?P<groupId>[^/]+)/expenses", financial.GetGroupExpensesHandler)
	router.AddRoute("POST", "/VassistantBackendProxy/financial/groups/(?P<groupId>[^/]+)/expenses", financial.PostGroupExpenseHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/groups/(?P<groupId>[^/]+)/users", financial.GetGroupUsersHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/expense-split-types", financial.GetExpenseSplitTypeHandler)
	router.AddRoute("GET", "/VassistantBackendProxy/financial/expense-categories", financial.GetExpenseCategoriesHandler)
}

func rootHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("Request path:", request.Path)
	log.Println("Request HTTP method:", request.HTTPMethod)
	return router.Serve(request)
}

func main() {
	lambda.Start(rootHandler)
}
