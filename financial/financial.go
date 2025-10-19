package financial

import (
	"encoding/json"
	"log"

	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Participant struct for financial expense participants
type Participant struct {
	UserID string      `json:"userId" dynamodbav:"userId"`
	Share  json.Number `json:"share" dynamodbav:"share"`
}

// FinancialExpense struct for the "get financial" response
type FinancialExpense struct {
	ExpenseID    string        `json:"expenseId" dynamodbav:"expenseId"`
	GroupID      string        `json:"groupId" dynamodbav:"groupId"`
	Description  string        `json:"description" dynamodbav:"description"`
	Category     string        `json:"category" dynamodbav:"category"`
	Amount       json.Number   `json:"amount" dynamodbav:"amount"`
	DateTime     string        `json:"dateTime" dynamodbav:"dateTime"`
	PaidBy       string        `json:"paidBy" dynamodbav:"paidBy"`
	ImageURL     string        `json:"imageUrl" dynamodbav:"imageUrl"`
	SplitType    string        `json:"splitType" dynamodbav:"splitType"`
	Participants []Participant `json:"participants" dynamodbav:"participants"`
	Settled      bool          `json:"settled" dynamodbav:"settled"`
}

// GroupMember struct for the splitter-group-members table
type GroupMember struct {
	UserID    string `json:"userId" dynamodbav:"userId"`
	GroupID   string `json:"groupId" dynamodbav:"groupId"`
	GroupName string `json:"groupName" dynamodbav:"groupName"`
}

var DynamoDbClient *dynamodb.Client

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

func GetGroupExpensesHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	// Extract groupId from path parameters
	groupId, ok := request.PathParameters["groupId"]
	if !ok || groupId == "" {
		return createErrorResponse(400, "Group ID is missing")
	}

	// Build the query input
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("splitter-expenses"),
		KeyConditionExpression: aws.String("groupId = :groupId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":groupId": &types.AttributeValueMemberS{Value: groupId},
		},
	}

	// Make the DynamoDB Query API call
	result, err := DynamoDbClient.Query(context.TODO(), queryInput)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Unmarshal the Items into a slice of FinancialExpense structs
	var expenses []FinancialExpense
	err = attributevalue.UnmarshalListOfMaps(result.Items, &expenses)
	if err != nil {
		log.Printf("Error unmarshalling expenses: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	log.Printf("Successfully retrieved %d expenses for group %s", len(expenses), groupId)

	// Marshal the expenses into JSON for the payload
	payload, err := json.Marshal(expenses)
	if err != nil {
		log.Println("Error marshalling expenses:", err)
		return createErrorResponse(500, "Internal server error")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(payload),
	}, nil
}

func GetGroupsHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	// Extract claims from the authorizer
	authorizer := request.RequestContext.Authorizer
	claims, ok := authorizer["claims"].(map[string]interface{})
	if !ok {
		log.Println("Error: Invalid claims format")
		return createErrorResponse(403, "Unauthorized: Invalid claims format")
	}
	sub, _ := claims["sub"].(string)

	// Build the query input
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("splitter-group-members"),
		KeyConditionExpression: aws.String("userId = :userId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: sub},
		},
		ProjectionExpression: aws.String("userId, groupId, groupName"),
	}

	// Make the DynamoDB Query API call
	result, err := DynamoDbClient.Query(context.TODO(), queryInput)
	if err != nil {
		log.Printf("Error querying DynamoDB: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Unmarshal the Items into a slice of GroupMember structs
	var groupMembers []GroupMember
	err = attributevalue.UnmarshalListOfMaps(result.Items, &groupMembers)
	if err != nil {
		log.Printf("Error unmarshalling group members: %v", err)
		return createErrorResponse(500, "Internal server error")
	}

	log.Printf("Successfully retrieved %d groups for user %s", len(groupMembers), sub)

	// Marshal the group members into JSON for the payload
	payload, err := json.Marshal(groupMembers)
	if err != nil {
		log.Println("Error marshalling group members:", err)
		return createErrorResponse(500, "Internal server error")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(payload),
	}, nil
}
