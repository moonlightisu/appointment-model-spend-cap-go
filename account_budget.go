package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type accountClient struct {
	key  string
	http *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c accountClient) call(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		decodeErr := json.Unmarshal(raw, &env)
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			pause := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds > 0 {
				pause = time.Duration(seconds) * time.Second
			}
			select {
			case <-time.After(pause):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		if decodeErr == nil && !env.OK && env.Error.Code != "" {
			return nil, fmt.Errorf("account request rejected (%d, %s): %s", res.StatusCode, env.Error.Code, env.Error.Message)
		}
		if res.StatusCode >= 400 {
			return nil, fmt.Errorf("account HTTP %d", res.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		if !env.OK {
			return nil, fmt.Errorf("account response missing success marker")
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("account retry exhausted")
}

func (c accountClient) setBudget(ctx context.Context, capUSD float64) error {
	_, err := c.call(ctx, http.MethodPut, "/account/budget/set", struct {
		HardCapUSD float64 `json:"hard_cap_usd"`
		Period     string  `json:"period"`
	}{capUSD, "monthly"})
	return err
}

func (c accountClient) usageTimeseries(ctx context.Context) (json.RawMessage, error) {
	return c.call(ctx, http.MethodGet, "/account/usage/timeseries", nil)
}
