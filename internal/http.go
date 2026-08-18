package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// HTTPClient 轻量 HTTP 客户端
type HTTPClient struct {
	c *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{c: &http.Client{Timeout: 25 * time.Second}}
}

// PostJSON 发送 POST JSON，返回 (http状态码, 响应体map, error)
func (h *HTTPClient) PostJSON(url string, headers map[string]string, body interface{}) (int, map[string]interface{}, error) {
	return h.do(url, headers, body)
}

// PostJSONRetry 带指数退避重试
func (h *HTTPClient) PostJSONRetry(url string, headers map[string]string, body interface{}, retries int) (int, map[string]interface{}, error) {
	var lastErr error
	for i := 0; i <= retries; i++ {
		code, resp, err := h.do(url, headers, body)
		if err == nil {
			return code, resp, nil
		}
		lastErr = err
		if i < retries {
			time.Sleep(time.Duration(i+1) * 3 * time.Second)
		}
	}
	return 0, nil, lastErr
}

func (h *HTTPClient) do(url string, headers map[string]string, body interface{}) (int, map[string]interface{}, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return 0, nil, err
		}
	}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := map[string]interface{}{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &out)
	}
	return resp.StatusCode, out, nil
}
