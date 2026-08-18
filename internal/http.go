package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// PostJSONRetry 带指数退避重试（仅网络错误重试；服务端已响应的 4xx/5xx 不重试）
func (h *HTTPClient) PostJSONRetry(url string, headers map[string]string, body interface{}, retries int) (int, map[string]interface{}, error) {
	var lastErr error
	var lastCode int
	var lastResp map[string]interface{}
	for i := 0; i <= retries; i++ {
		code, resp, err := h.do(url, headers, body)
		if err == nil {
			return code, resp, nil
		}
		lastCode, lastResp, lastErr = code, resp, err
		if code >= 400 {
			return code, resp, err
		}
		if i < retries {
			time.Sleep(time.Duration(i+1) * 3 * time.Second)
		}
	}
	return lastCode, lastResp, lastErr
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
		if err := json.Unmarshal(data, &out); err != nil {
			return resp.StatusCode, out, fmt.Errorf("响应不是合法 JSON: %s", truncate(string(data), 200))
		}
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, out, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
