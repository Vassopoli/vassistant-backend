package financial

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"vassistant-backend/common"
)
// MockDynamoDBClient is a mock implementation of the DynamoDBAPI interface
type MockDynamoDBClient struct {
	common.DynamoDBAPI
	QueryFunc func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	// Not needed for these tests
	return nil, nil
}

func (m *MockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.QueryFunc(ctx, params, optFns...)
}

func TestGetGroupsHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		QueryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			// Create a sample group member
			groupMember := GroupMember{
				UserID:    "test-user-id",
				GroupID:   "test-group-id",
				GroupName: "Test Group",
			}
			// Marshal the group member into a DynamoDB attribute value map
			av, err := attributevalue.MarshalMap(groupMember)
			if err != nil {
				return nil, err
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{av},
				Count: 1,
			}, nil
		},
	}
	DynamoDbClient = mockClient

	// Create a sample request
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"claims": map[string]interface{}{
					"sub": "test-user-id",
				},
			},
		},
	}

	// Call the handler
	response, err := GetGroupsHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var groupMembers []GroupMember
	err = json.Unmarshal([]byte(response.Body), &groupMembers)
	assert.NoError(t, err)
	assert.Len(t, groupMembers, 1)

	// Verify the group member
	groupMember := groupMembers[0]
	assert.Equal(t, "test-user-id", groupMember.UserID)
	assert.Equal(t, "test-group-id", groupMember.GroupID)
	assert.Equal(t, "Test Group", groupMember.GroupName)
}
