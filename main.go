package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	Content  string `json:"content"`
}

// IncomingRequest struct to parse the request body
type IncomingRequest struct {
	Content string `json:"content"`
}

func postMessageHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v\n", request)

	targetAPI := os.Getenv("TARGET_API_HOSTNAME")
	if targetAPI == "" {
		log.Println("Error: TARGET_API_HOSTNAME environment variable not set.")
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
		Content:  incomingReq.Content,
	}

	// Marshal the message into JSON for the payload
	payload, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Create a new HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("POST", targetAPI, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Error creating HTTP request:", err)
		return createErrorResponse(500, "Internal server error")
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending HTTP request:", err)
		return createErrorResponse(500, "Error sending request to target API")
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return createErrorResponse(500, "Internal server error")
	}

	// Return the response from the target API
	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
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
		Content:  "Mock",
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
