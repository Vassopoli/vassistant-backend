package messages

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"vassistant-backend/common"
)

// MockDynamoDBClient is a mock implementation of the DynamoDBAPI interface
type MockDynamoDBClient struct {
	common.DynamoDBAPI
	PutItemFunc func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.PutItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	// Not needed for this test
	return nil, nil
}

func TestPostMessageHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	DynamoDbClient = mockClient

	// Create a sample request
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"claims": map[string]interface{}{
					"sub":              "test-user-id",
					"cognito:username": "test-user",
				},
			},
		},
		Body: `{"content": "Hello, world!"}`,
	}

	// Call the handler
	response, err := PostMessageHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var responseMessages []GetMessage
	err = json.Unmarshal([]byte(response.Body), &responseMessages)
	assert.NoError(t, err)
	assert.Len(t, responseMessages, 2)

	// Verify the user message
	userMessage := responseMessages[0]
	assert.Equal(t, "test-user-id", userMessage.UserId)
	assert.Equal(t, "test-user", userMessage.Username)
	assert.Equal(t, "user", userMessage.Role)
	assert.Equal(t, "Hello, world!", userMessage.Content)

	// Verify the assistant message
	assistantMessage := responseMessages[1]
	assert.Equal(t, "test-user-id", assistantMessage.UserId)
	assert.Equal(t, "ai-assistant", assistantMessage.Username)
	assert.Equal(t, "assistant", assistantMessage.Role)
	assert.Equal(t, "This is a mock response from the assistant.", assistantMessage.Content)
}
