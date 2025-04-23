package headscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	URL       *url.URL
	apiKey    string
	UserAgent string
	HTTP      *http.Client
	Logger    *slog.Logger
}

type HeadscaleClient interface {
	PreAuthKeys() *PreAuthKeyClient
	do(ctx context.Context, req *http.Request, v any) error
	buildRequest(ctx context.Context, method string, uri *url.URL, req request) (*http.Request, error)
	buildPath(parts ...string) *url.URL
}

func (c *Client) PreAuthKeys() *PreAuthKeyClient {
	return &PreAuthKeyClient{
		client: c,
	}
}

func (c *Client) buildPath(parts ...string) *url.URL {
	parts = append([]string{basePath}, parts...)
	return c.URL.JoinPath(parts...)
}

const (
	DefaultUserAgent string        = "tailscale-sidecar-injector"
	DefaultTimeout   time.Duration = 5 * time.Second
	basePath                       = "/api/v1"
)

func New(ctx context.Context, apiKey, address string) (*Client, error) {
	res := &Client{}

	res.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	if address == "" {
		address = os.Getenv("HEADSCALE_CLI_ADDRESS")
	}

	if u, err := url.Parse(address); err != nil {
		return nil, err
	} else {
		res.URL = u
	}

	if apiKey == "" {
		res.apiKey = os.Getenv("HEADSCALE_CLI_API_KEY")
	} else {
		res.apiKey = apiKey
	}

	res.UserAgent = DefaultUserAgent

	res.HTTP = &http.Client{
		Timeout: DefaultTimeout,
	}

	return res, nil
}

type request struct {
	body        any
	headers     map[string]string
	contentType string
	params      map[string]string
}

func (c *Client) do(ctx context.Context, req *http.Request, v any) error {
	c.Logger.Debug("making http request", "method", req.Method, "url", req.URL.String(), "query", req.URL.RawQuery)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		c.Logger.ErrorContext(ctx, "failed making the request", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.Logger.ErrorContext(ctx, "unexpected status code", "status", resp.StatusCode)
		var b []byte
		if n, err := resp.Body.Read(b); err == nil {
			if n > 0 {
				c.Logger.DebugContext(ctx, "response", "body", string(b))
			}
		}
		return fmt.Errorf("unexepcted status code %d", resp.StatusCode)
	}
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) buildRequest(ctx context.Context, method string, uri *url.URL, req request) (*http.Request, error) {
	var bodyBytes []byte
	if req.body != nil {
		var err error
		switch body := req.body.(type) {
		case []byte:
			bodyBytes = body
		case string:
			bodyBytes = []byte(body)
		default:
			bodyBytes, err = json.Marshal(req.body)
			if err != nil {
				return nil, err
			}
		}
	}
	query := uri.Query()
	for k, v := range req.params {
		query.Add(k, fmt.Sprint(v))
	}
	uri.RawQuery = query.Encode()

	r, err := http.NewRequestWithContext(ctx, method, uri.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", c.UserAgent)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	if req.contentType != "" {
		r.Header.Set("Content-Type", req.contentType)
	}

	for k, v := range req.headers {
		r.Header.Set(k, v)
	}
	return r, nil
}
