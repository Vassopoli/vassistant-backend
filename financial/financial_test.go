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

func TestGetGroupUsersHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		QueryFunc: func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			// Create sample group members
			groupMember1 := GroupMember{
				UserID: "user-1",
			}
			groupMember2 := GroupMember{
				UserID: "user-2",
			}

			// Marshal the group members
			av1, err := attributevalue.MarshalMap(groupMember1)
			if err != nil {
				return nil, err
			}
			av2, err := attributevalue.MarshalMap(groupMember2)
			if err != nil {
				return nil, err
			}

			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{av1, av2},
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
	response, err := GetGroupUsersHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var users []User
	err = json.Unmarshal([]byte(response.Body), &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Verify the users
	assert.Equal(t, "user-1", users[0].UserID)
	assert.Equal(t, "User One", users[0].ShowableName)
	assert.Equal(t, "user-2", users[1].UserID)
	assert.Equal(t, "User Two", users[1].ShowableName)
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
				Title: "Older Expense",
				DateTime:    "2023-01-01T00:00:00Z",
				Amount:      "100",
				PaidBy:      "user-1",
				CreatedBy:     "user-3",
				Participants: []Participant{
					{UserID: "user-1", Share: "50"},
					{UserID: "user-2", Share: "50"},
				},
			}
			expense2 := FinancialExpense{
				ExpenseID:   "test-expense-2",
				GroupID:     "test-group-id",
				Title: "Newer Expense",
				DateTime:    "2023-01-02T00:00:00Z",
				Amount:      "200",
				PaidBy:      "user-2",
				CreatedBy:     "user-3",
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
	assert.Equal(t, "user-3", expense1.CreatedBy)
	assert.Equal(t, "User Three", expense1.CreatedByUser.ShowableName)
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
	testDateTime := "2024-01-02T15:04:05Z"
	expense := FinancialExpense{
		Title: "Test Expense",
		Amount:      "100",
		DateTime:    testDateTime,
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
	assert.Equal(t, "test-user-id", createdExpense.CreatedBy)
	assert.NotEmpty(t, createdExpense.CreatedAt)
	assert.Equal(t, testDateTime, createdExpense.DateTime)

	// Check if CreatedAt is a valid timestamp
	_, err = time.Parse(time.RFC3339, createdExpense.CreatedAt)
	assert.NoError(t, err)
}

func TestGetExpenseCategoriesHandler(t *testing.T) {
	// Create a sample request
	request := events.APIGatewayProxyRequest{}

	// Call the handler
	response, err := GetExpenseCategoriesHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var categories []string
	err = json.Unmarshal([]byte(response.Body), &categories)
	assert.NoError(t, err)
	assert.Len(t, categories, 1)
	assert.Equal(t, "FOOD", categories[0])
}

func TestGetExpenseHandler(t *testing.T) {
	// Set up the mock DynamoDB client
	mockClient := &MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			// Create a sample expense
			expense := FinancialExpense{
				ExpenseID: "test-expense-id",
				GroupID:   "test-group-id",
				PaidBy:    "user-1",
				CreatedBy:   "user-2",
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
			return &dynamodb.GetItemOutput{
				Item: av,
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
			"groupId":   "test-group-id",
			"expenseId": "test-expense-id",
		},
	}

	// Call the handler
	response, err := GetExpenseHandler(request)
	assert.NoError(t, err)

	// Check the response
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var fetchedExpense FinancialExpense
	err = json.Unmarshal([]byte(response.Body), &fetchedExpense)
	assert.NoError(t, err)

	// Verify the expense details
	assert.Equal(t, "test-expense-id", fetchedExpense.ExpenseID)
	assert.Equal(t, "test-group-id", fetchedExpense.GroupID)
	assert.Equal(t, "User One", fetchedExpense.PaidByUser.ShowableName)
	assert.Equal(t, "User Two", fetchedExpense.CreatedByUser.ShowableName)
}

func TestGetExpenseHandlerNotFound(t *testing.T) {
	// Set up the mock DynamoDB client to return not found
	mockClient := &MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: nil, // Simulate not found
			}, nil
		},
	}
	DynamoDbClient = mockClient

	// Create a sample request
	request := events.APIGatewayProxyRequest{
		PathParameters: map[string]string{
			"groupId":   "test-group-id",
			"expenseId": "non-existent-expense-id",
		},
	}

	// Call the handler
	response, err := GetExpenseHandler(request)
	assert.NoError(t, err)

	// Check the response for 404 Not Found
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}
