package financial

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

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
	PutItemFunc      func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	return m.BatchGetItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.PutItemFunc(ctx, params, optFns...)
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
			// Assert that the query is asking for descending order
			assert.NotNil(t, params.ScanIndexForward)
			assert.False(t, *params.ScanIndexForward)

			// Create sample expenses
			expense1 := FinancialExpense{
				ExpenseID:   "test-expense-1",
				GroupID:     "test-group-id",
				Description: "Older Expense",
				DateTime:    "2023-01-01T00:00:00Z",
				Amount:      "100",
				PaidBy:      "user-1",
				AddedBy:     "user-3",
				Participants: []Participant{
					{UserID: "user-1", Share: "50"},
					{UserID: "user-2", Share: "50"},
				},
			}
			expense2 := FinancialExpense{
				ExpenseID:   "test-expense-2",
				GroupID:     "test-group-id",
				Description: "Newer Expense",
				DateTime:    "2023-01-02T00:00:00Z",
				Amount:      "200",
				PaidBy:      "user-2",
				AddedBy:     "user-3",
				Participants: []Participant{
					{UserID: "user-1", Share: "100"},
					{UserID: "user-2", Share: "100"},
				},
			}

			// Marshal the expenses
			av1, err := attributevalue.MarshalMap(expense1)
			if err != nil {
				return nil, err
			}
			av2, err := attributevalue.MarshalMap(expense2)
			if err != nil {
				return nil, err
			}

			// Return the items as DynamoDB would: sorted descending
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{av2, av1},
				Count: 2,
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
			user3 := map[string]types.AttributeValue{
				"userId":       &types.AttributeValueMemberS{Value: "user-3"},
				"showableName": &types.AttributeValueMemberS{Value: "User Three"},
			}
			return &dynamodb.BatchGetItemOutput{
				Responses: map[string][]map[string]types.AttributeValue{
					"vassistant-users": {user1, user2, user3},
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
	assert.Len(t, expenses, 2)

	// Verify that the handler returns the expenses in the order they were received
	assert.Equal(t, "test-expense-2", expenses[0].ExpenseID)
	assert.Equal(t, "test-expense-1", expenses[1].ExpenseID)

	// Verify the first expense's details
	expense1 := expenses[0]
	assert.Equal(t, "user-2", expense1.PaidBy)
	assert.Equal(t, "User Two", expense1.PaidByUser.ShowableName)
	assert.Equal(t, "user-3", expense1.AddedBy)
	assert.Equal(t, "User Three", expense1.AddedByUser.ShowableName)
	assert.Len(t, expense1.Participants, 2)
	assert.Equal(t, "user-1", expense1.Participants[0].UserID)
	assert.Equal(t, "User One", expense1.Participants[0].User.ShowableName)
	assert.Equal(t, "user-2", expense1.Participants[1].UserID)
	assert.Equal(t, "User Two", expense1.Participants[1].User.ShowableName)

	// Verify the second expense's details
	expense2 := expenses[1]
	assert.Equal(t, "user-1", expense2.PaidBy)
	assert.Equal(t, "User One", expense2.PaidByUser.ShowableName)
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

func TestPostGroupExpenseHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	DynamoDbClient = mockClient

	// Create a sample request body
	expense := FinancialExpense{
		Description: "Test Expense",
		Amount:      "100",
		PaidBy:      "user-1",
		Participants: []Participant{
			{UserID: "user-1", Share: "50"},
			{UserID: "user-2", Share: "50"},
		},
	}
	body, err := json.Marshal(expense)
	assert.NoError(t, err)

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
		Body: string(body),
	}

	// Call the handler
	response, err := PostGroupExpenseHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var createdExpense FinancialExpense
	err = json.Unmarshal([]byte(response.Body), &createdExpense)
	assert.NoError(t, err)

	// Verify the created expense
	assert.NotEmpty(t, createdExpense.ExpenseID)
	assert.Equal(t, "test-group-id", createdExpense.GroupID)
	assert.Equal(t, "test-user-id", createdExpense.AddedBy)
	assert.NotEmpty(t, createdExpense.AddedAt)

	// Check if AddedAt is a valid timestamp
	_, err = time.Parse(time.RFC3339, createdExpense.AddedAt)
	assert.NoError(t, err)
}
