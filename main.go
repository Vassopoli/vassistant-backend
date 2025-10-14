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
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal server configuration error"}, nil
	}

	// Extract claims from the authorizer
	authorizer := request.RequestContext.Authorizer
	claims, ok := authorizer["claims"].(map[string]interface{})
	if !ok {
		return events.APIGatewayProxyResponse{StatusCode: 403, Body: "Unauthorized: Invalid claims format"}, nil
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["cognito:username"].(string)

	// Parse the incoming request body
	var incomingReq IncomingRequest
	err := json.Unmarshal([]byte(request.Body), &incomingReq)
	if err != nil {
		log.Println("Error unmarshalling request body:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request body format",
		}, nil
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
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal server error",
		}, nil
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
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error invoking target function"}, nil
	}

	if result.FunctionError != nil {
		log.Printf("Error returned by target lambda function: %s", *result.FunctionError)
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Target function returned an error"}, nil
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
		return events.APIGatewayProxyResponse{StatusCode: 403, Body: "Unauthorized: Invalid claims format"}, nil
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["cognito:username"].(string)

	// Create the outgoing message payload
	message := Message{
		Id:       sub,
		Username: username,
		Text:     "Mock",
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(message),
	}, nil
}

func rootHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.Path == "/messages" {
		switch request.HTTPMethod {
		case "POST":
			return postMessageHandler(request)
		case "GET":
			return getMessageHandler(request)
		default:
			return events.APIGatewayProxyResponse{StatusCode: 405, Body: "Method Not Allowed"}, nil
		}
	}
	return events.APIGatewayProxyResponse{StatusCode: 404, Body: "Not Found"}, nil
}

func main() {
	lambda.Start(rootHandler)
}
