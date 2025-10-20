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
	QueryFunc        func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	BatchGetItemFunc func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	GetItemFunc      func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	return m.BatchGetItemFunc(ctx, params, optFns...)
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

func TestGetGroupExpensesHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		QueryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			// Create a sample expense
			expense := FinancialExpense{
				ExpenseID:   "test-expense-id",
				GroupID:     "test-group-id",
				Description: "Test Expense",
				Amount:      "100",
				PaidBy:      "user-1",
				Participants: []Participant{
					{UserID: "user-1", Share: "50"},
					{UserID: "user-2", Share: "50"},
				},
			}
			// Marshal the expense into a DynamoDB attribute value map
			av, err := attributevalue.MarshalMap(expense)
			if err != nil {
				return nil, err
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{av},
				Count: 1,
			}, nil
		},
		BatchGetItemFunc: func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
			// Create sample user data
			user1 := map[string]types.AttributeValue{
				"userId":       &types.AttributeValueMemberS{Value: "user-1"},
				"showableName": &types.AttributeValueMemberS{Value: "User One"},
			}
			user2 := map[string]types.AttributeValue{
				"userId":       &types.AttributeValueMemberS{Value: "user-2"},
				"showableName": &types.AttributeValueMemberS{Value: "User Two"},
			}
			return &dynamodb.BatchGetItemOutput{
				Responses: map[string][]map[string]types.AttributeValue{
					"vassistant-users": {user1, user2},
				},
			}, nil
		},
	}
	DynamoDbClient = mockClient

	// Create a sample request
	request := events.APIGatewayProxyRequest{
		PathParameters: map[string]string{
			"groupId": "test-group-id",
		},
	}

	// Call the handler
	response, err := GetGroupExpensesHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var expenses []FinancialExpense
	err = json.Unmarshal([]byte(response.Body), &expenses)
	assert.NoError(t, err)
	assert.Len(t, expenses, 1)

	// Verify the expense and participants
	expense := expenses[0]
	assert.Equal(t, "test-expense-id", expense.ExpenseID)
	assert.Equal(t, "user-1", expense.PaidBy)
	assert.Equal(t, "User One", expense.PaidByUser.ShowableName)
	assert.Len(t, expense.Participants, 2)
	assert.Equal(t, "user-1", expense.Participants[0].UserID)
	assert.Equal(t, "User One", expense.Participants[0].User.ShowableName)
	assert.Equal(t, "user-2", expense.Participants[1].UserID)
	assert.Equal(t, "User Two", expense.Participants[1].User.ShowableName)
}

func TestGetGroupHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
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
			return &dynamodb.GetItemOutput{
				Item: av,
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
		PathParameters: map[string]string{
			"groupId": "test-group-id",
		},
	}

	// Call the handler
	response, err := GetGroupHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var groupMember GroupMember
	err = json.Unmarshal([]byte(response.Body), &groupMember)
	assert.NoError(t, err)

	// Verify the group member
	assert.Equal(t, "test-user-id", groupMember.UserID)
	assert.Equal(t, "test-group-id", groupMember.GroupID)
	assert.Equal(t, "Test Group", groupMember.GroupName)
}
