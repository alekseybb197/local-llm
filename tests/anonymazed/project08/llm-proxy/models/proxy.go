package models

import (
	"bytes"
	"io"
	"net/http"
)

type ProxyRequest struct {
	OriginalRequest *http.Request
	Body            []byte
}

type ProxyResponse struct {
	StatusCode int
	Body        []byte
	Headers     http.Header
}

func (r *ProxyRequest) BodyToString() string {
	return string(r.Body)
}

func ReadRequestBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}

func CreateProxyRequest(originalRequest *http.Request) (*ProxyRequest, error) {
	body, err := io.ReadAll(originalRequest.Body)
	if err != nil {
		return nil, err
	}
	defer originalRequest.Body.Close()

	return &ProxyRequest{
		OriginalRequest: originalRequest,
		Body:            body,
	}, nil
}

func CreateProxyResponse(statusCode int, body []byte, headers http.Header) *ProxyResponse {
	return &ProxyResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    headers,
	}
}

func (r *ProxyResponse) ToHTTPResponse() (*http.Response, error) {
	resp := &http.Response{
		Status:     http.StatusText(r.StatusCode),
		StatusCode: r.StatusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(r.Body)),
	}

	for key, values := range r.Headers {
		for _, value := range values {
			resp.Header.Add(key, value)
		}
	}

	return resp, nil
}
