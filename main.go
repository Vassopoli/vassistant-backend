package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

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
	Text     string `json:"text"`
}

// IncomingRequest struct to parse the request body
type IncomingRequest struct {
	Text string `json:"text"`
}

var lambdaClient *awslambda.Client

func init() {
	// Initialize AWS SDK config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	// Initialize Lambda client
	lambdaClient = awslambda.NewFromConfig(cfg)
}

func postMessageHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	targetLambda := os.Getenv("TARGET_LAMBDA_FUNCTION_NAME")
	if targetLambda == "" {
		log.Println("Error: TARGET_LAMBDA_FUNCTION_NAME environment variable not set.")
		return createErrorResponse(500, "Internal server configuration error")
	}

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

	// Create the outgoing message payload
	message := Message{
		Id:       sub,
		Username: username,
		Text:     incomingReq.Text,
	}

	// Marshal the message into JSON for the payload
	payload, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Prepare the input for the Lambda invocation
	invokeInput := &awslambda.InvokeInput{
		FunctionName:   &targetLambda,
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	}

	// Invoke the target Lambda function
	result, err := lambdaClient.Invoke(context.TODO(), invokeInput)
	if err != nil {
		log.Println("Error invoking target lambda function:", err)
		return createErrorResponse(500, "Error invoking target function")
	}

	if result.FunctionError != nil {
		log.Printf("Error returned by target lambda function: %s", *result.FunctionError)
		return createErrorResponse(500, "Target function returned an error")
	}

	// Return the response from the invoked function
	return events.APIGatewayProxyResponse{
		StatusCode: int(result.StatusCode),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(result.Payload),
	}, nil
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

	// Create the outgoing message payload
	message := Message{
		Id:       sub,
		Username: username,
		Text:     "Mock",
	}

	// Marshal the message into JSON for the payload
	payload, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
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

	if request.Path == "/messages" {
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
