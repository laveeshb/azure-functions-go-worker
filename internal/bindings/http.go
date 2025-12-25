package bindings

import (
	"net/http"
	"strconv"
	"strings"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

// HttpRequest represents an HTTP request in a user-friendly format.
type HttpRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Query   map[string]string
	Params  map[string]string
	Body    []byte
	RawBody []byte
}

// HttpResponse represents an HTTP response to be sent back.
type HttpResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{} // Can be string, []byte, or any JSON-serializable value
}

// NewHttpRequest creates an HttpRequest from protobuf RpcHttp.
func NewHttpRequest(rpcHttp *pb.RpcHttp) *HttpRequest {
	if rpcHttp == nil {
		return &HttpRequest{
			Headers: make(map[string]string),
			Query:   make(map[string]string),
			Params:  make(map[string]string),
		}
	}

	req := &HttpRequest{
		Method:  rpcHttp.Method,
		URL:     rpcHttp.Url,
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Params:  make(map[string]string),
	}

	// Copy headers
	for k, v := range rpcHttp.Headers {
		req.Headers[strings.ToLower(k)] = v
	}

	// Copy query parameters
	for k, v := range rpcHttp.Query {
		req.Query[k] = v
	}

	// Copy route parameters
	for k, v := range rpcHttp.Params {
		req.Params[k] = v
	}

	// Extract body
	if rpcHttp.Body != nil {
		req.Body = TypedDataToBytes(rpcHttp.Body)
	}

	// Extract raw body
	if rpcHttp.RawBody != nil {
		req.RawBody = TypedDataToBytes(rpcHttp.RawBody)
	}

	return req
}

// GetQuery returns a query parameter value, or empty string if not found.
func (r *HttpRequest) GetQuery(key string) string {
	if r.Query == nil {
		return ""
	}
	return r.Query[key]
}

// GetHeader returns a header value (case-insensitive), or empty string if not found.
func (r *HttpRequest) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers[strings.ToLower(key)]
}

// GetParam returns a route parameter value, or empty string if not found.
func (r *HttpRequest) GetParam(key string) string {
	if r.Params == nil {
		return ""
	}
	return r.Params[key]
}

// BodyAsString returns the request body as a string.
func (r *HttpRequest) BodyAsString() string {
	return string(r.Body)
}

// ToRpcHttp converts HttpResponse to protobuf RpcHttp.
func (r *HttpResponse) ToRpcHttp() (*pb.RpcHttp, error) {
	rpcHttp := &pb.RpcHttp{
		StatusCode: strconv.Itoa(r.StatusCode),
		Headers:    make(map[string]string),
	}

	// Copy headers
	for k, v := range r.Headers {
		rpcHttp.Headers[k] = v
	}

	// Set Content-Type if not already set
	if _, hasContentType := rpcHttp.Headers["Content-Type"]; !hasContentType {
		rpcHttp.Headers["Content-Type"] = "text/plain"
	}

	// Convert body
	switch body := r.Body.(type) {
	case string:
		rpcHttp.Body = StringToTypedData(body)
	case []byte:
		rpcHttp.Body = BytesToTypedData(body)
	case nil:
		// No body
	default:
		// Try to serialize as JSON
		typedData, err := JSONToTypedData(body)
		if err != nil {
			return nil, err
		}
		rpcHttp.Body = typedData
		rpcHttp.Headers["Content-Type"] = "application/json"
	}

	return rpcHttp, nil
}

// NewHttpResponse creates a new HttpResponse with default values.
func NewHttpResponse() *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
	}
}

// OK creates an HTTP 200 response with the given body.
func OK(body interface{}) *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
		Body:       body,
	}
}

// Created creates an HTTP 201 response with the given body.
func Created(body interface{}) *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusCreated,
		Headers:    make(map[string]string),
		Body:       body,
	}
}

// BadRequest creates an HTTP 400 response with the given message.
func BadRequest(message string) *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusBadRequest,
		Headers:    make(map[string]string),
		Body:       message,
	}
}

// NotFound creates an HTTP 404 response with the given message.
func NotFound(message string) *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusNotFound,
		Headers:    make(map[string]string),
		Body:       message,
	}
}

// InternalServerError creates an HTTP 500 response with the given message.
func InternalServerError(message string) *HttpResponse {
	return &HttpResponse{
		StatusCode: http.StatusInternalServerError,
		Headers:    make(map[string]string),
		Body:       message,
	}
}

// WithHeader adds a header to the response and returns the response for chaining.
func (r *HttpResponse) WithHeader(key, value string) *HttpResponse {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// WithContentType sets the Content-Type header and returns the response for chaining.
func (r *HttpResponse) WithContentType(contentType string) *HttpResponse {
	return r.WithHeader("Content-Type", contentType)
}
