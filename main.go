package main

import (
	"encoding/json"
	"log"
	"time"

	"context"

	"github.com/google/uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dynamoDbClient *dynamodb.Client

func init() {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create DynamoDB client
	dynamoDbClient = dynamodb.NewFromConfig(cfg)
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

// Message struct for the outgoing payload
type Message struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Content  string `json:"content"`
}

// GetMessage struct for the "get messages" response
type GetMessage struct {
	Id        string `json:"id" dynamodbav:"id"`
	UserId    string `json:"userId" dynamodbav:"userId"`
	Username  string `json:"username" dynamodbav:"username"`
	Role      string `json:"role" dynamodbav:"role"`
	Content   string `json:"content" dynamodbav:"content"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
}

// IncomingRequest struct to parse the request body
type IncomingRequest struct {
	Content string `json:"content"`
}

func postMessageHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	// Extract claims from the authorizer
	authorizer := request.RequestContext.Authorizer
	claims, ok := authorizer["claims"].(map[string]interface{})
	if !ok {
		log.Println("Error: Invalid claims format")
		return createErrorResponse(403, "Unauthorized: Invalid claims format")
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["cognito:username"].(string)

	// Parse the incoming request body
	var incomingReq IncomingRequest
	err := json.Unmarshal([]byte(request.Body), &incomingReq)
	if err != nil {
		log.Println("Error unmarshalling request body:", err)
		return createErrorResponse(400, "Invalid request body format")
	}

	// Create the new message object
	newMessage := GetMessage{
		Id:        uuid.New().String(),
		UserId:    sub,
		Username:  username,
		Role:      "user",
		Content:   incomingReq.Content,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Marshal the message into an attribute value map
	av, err := attributevalue.MarshalMap(newMessage)
	if err != nil {
		log.Printf("Error marshalling message to AttributeValue: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Create the PutItem input
	putItemInput := &dynamodb.PutItemInput{
		TableName: aws.String("chat"),
		Item:      av,
	}

	// Save the message to DynamoDB
	_, err = dynamoDbClient.PutItem(context.TODO(), putItemInput)
	if err != nil {
		log.Printf("Error saving message to DynamoDB: %v", err)
		return createErrorResponse(500, "Failed to save message")
	}

	// Save a mock assistant message
	assistantMessage, err := saveAssistantMessage(sub)
	if err != nil {
		log.Printf("Error saving assistant message to DynamoDB: %v", err)
		return createErrorResponse(500, "Failed to save assistant message")
	}

	// Create a response that includes both the user's message and the assistant's message
	responseMessages := []GetMessage{newMessage, assistantMessage}

	// Marshal the messages into JSON for the response body
	responseBody, err := json.Marshal(responseMessages)
	if err != nil {
		log.Printf("Error marshalling response body: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Return a 201 Created response
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(responseBody),
	}, nil
}

func saveAssistantMessage(sub string) (GetMessage, error) {
	assistantMessage := GetMessage{
		Id:        uuid.New().String(),
		UserId:    sub,
		Username:  "ai-assistant",
		Role:      "assistant",
		Content:   "This is a mock response from the assistant.",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(assistantMessage)
	if err != nil {
		log.Printf("Error marshalling assistant message to AttributeValue: %v", err)
		return GetMessage{}, err
	}

	putItemInput := &dynamodb.PutItemInput{
		TableName: aws.String("chat"),
		Item:      av,
	}

	_, err = dynamoDbClient.PutItem(context.TODO(), putItemInput)
	if err != nil {
		log.Printf("Error saving assistant message to DynamoDB: %v", err)
		return GetMessage{}, err
	}
	return assistantMessage, nil
}

// queryMessagesByUserID queries the DynamoDB table for messages by userId
func queryMessagesByUserID(userID string) ([]GetMessage, error) {
	// Build the query input
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("chat"),
		KeyConditionExpression: aws.String("userId = :userId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(true), // Sort by createdAt ascending
	}

	// Make the DynamoDB Query API call
	result, err := dynamoDbClient.Query(context.TODO(), queryInput)
	if err != nil {
		return nil, err
	}

	// Unmarshal the Items into a slice of GetMessage structs
	var messages []GetMessage
	err = attributevalue.UnmarshalListOfMaps(result.Items, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func getMessageHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	// Extract claims from the authorizer
	authorizer := request.RequestContext.Authorizer
	claims, ok := authorizer["claims"].(map[string]interface{})
	if !ok {
		log.Println("Error: Invalid claims format")
		return createErrorResponse(403, "Unauthorized: Invalid claims format")
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["cognito:username"].(string)
	log.Printf("request from user: %s, sub: %s\n", username, sub)

	// Query messages from DynamoDB
	messages, err := queryMessagesByUserID(sub)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Marshal the messages into JSON for the payload
	payload, err := json.Marshal(messages)
	if err != nil {
		log.Println("Error marshalling messages:", err)
		return createErrorResponse(500, "Internal server error")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(payload),
	}, nil
}

func rootHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("Request path:", request.Path)
	log.Println("Request HTTP method:", request.HTTPMethod)

	if request.Path == "/VassistantBackendProxy/messages" {
		switch request.HTTPMethod {
		case "POST":
			return postMessageHandler(request)
		case "GET":
			return getMessageHandler(request)
		default:
			return createErrorResponse(405, "Method Not Allowed")
		}
	}
	return createErrorResponse(404, "Not Found")
}

func main() {
	lambda.Start(rootHandler)
}
