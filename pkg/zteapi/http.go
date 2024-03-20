package zteapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newError(resp *http.Response) *httpError {
	out := &httpError{
		Code: resp.StatusCode,
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err == nil {
		out.Message = string(body)
	} else {
		out.Message = err.Error()
	}
	return out
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http error: %d %s", e.Code, e.Message)
}

func httpDoRetry(req *http.Request, retries int, out interface{}) (*http.Response, error) {
	resp, err := httpDo(req, out)
	if err != nil && retries > 0 {
		return httpDoRetry(req, retries-1, out)
	}

	return resp, err
}

func httpDo(req *http.Request, out interface{}) (*http.Response, error) {
	referer := *req.URL
	referer.Path = "/index.html"
	req.Header.Add("Referer", referer.String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp, newError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return resp, err
	}

	return resp, nil
}
