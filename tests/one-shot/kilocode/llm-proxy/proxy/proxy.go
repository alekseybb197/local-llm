package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ProxyResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

func ProxyRequest(targetURL string, originalRequest *http.Request, headers string, timeout time.Duration) (*ProxyResponse, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Prepare request
	req := originalRequest.Clone(originalRequest.Context())
	req.URL = target
	req.Host = target.Host

	// Read body
	var bodyReader io.Reader
	if originalRequest.Body != nil {
		bodyBytes, err := io.ReadAll(originalRequest.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	originalRequest.Body = io.NopCloser(bodyReader)

	// Copy headers
	for k, v := range originalRequest.Header {
		req.Header[k] = v
	}

	// Add custom headers
	if headers != "" {
		for _, h := range splitHeaders(headers) {
			key, value := h[0], h[1]
			req.Header.Set(key, value)
		}
	}

	// Create client
	client := &http.Client{
		Timeout: timeout,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}, nil
}

func splitHeaders(s string) [][]string {
	entries := bytes.Fields([]byte(s))
	result := make([][]string, 0, len(entries))

	for _, entry := range entries {
		if bytes.Contains(entry, []byte(":")) {
			parts := bytes.SplitN(entry, []byte(":"), 2)
			result = append(result, make([]string, 2, 2))
			copy(result[len(result)-1], []string{string(parts[0]), string(parts[1])})
		}
	}

	return result
}
