package api

import (
	"regexp"
	"vassistant-backend/common"

	"github.com/aws/aws-lambda-go/events"
)

// HandlerFunc defines the function signature for our Lambda handlers.
type HandlerFunc func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Route defines the structure for a single API route.
type Route struct {
	Method  string
	Path    *regexp.Regexp
	Handler HandlerFunc
}

// Router is a collection of routes that can be served.
type Router struct {
	routes []Route
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{}
}

// AddRoute adds a new route to the router.
func (r *Router) AddRoute(method, path string, handler HandlerFunc) {
	route := Route{
		Method:  method,
		Path:    regexp.MustCompile("^" + path + "$"),
		Handler: handler,
	}
	r.routes = append(r.routes, route)
}

// Serve handles the incoming request by finding the appropriate route.
func (r *Router) Serve(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	for _, route := range r.routes {
		if route.Method == request.HTTPMethod {
			matches := route.Path.FindStringSubmatch(request.Path)
			if len(matches) > 0 {
				// Extract path parameters
				pathParams := make(map[string]string)
				for i, name := range route.Path.SubexpNames() {
					if i != 0 && name != "" {
						pathParams[name] = matches[i]
					}
				}
				request.PathParameters = pathParams
				return route.Handler(request)
			}
		}
	}
	// No matching route found
	return common.CreateErrorResponse(404, "Not Found")
}
