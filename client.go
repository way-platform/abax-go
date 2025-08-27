package abax

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
)

// Client to the Abax management APIs.
type Client struct {
	config ClientConfig
}

// NewClient creates a new Abax API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	client := Client{
		config: newClientConfig(),
	}
	for _, opt := range opts {
		opt(&client.config)
	}
	_ = client.newRequest // TODO: Remove this after adding first endpoint.
	return &client, nil
}

// ClientConfig configures a [Client].
type ClientConfig struct {
	refreshToken string
}

func newClientConfig() ClientConfig {
	return ClientConfig{}
}

// ClientOption is a configuration option for a [Client].
type ClientOption func(*ClientConfig)

// WithRefreshToken sets the refresh token for the client.
func WithRefreshToken(refreshToken string) ClientOption {
	return func(config *ClientConfig) {
		config.refreshToken = refreshToken
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	method string,
	requestPath string, // TODO: Add base URL to client config.
	body io.Reader,
) (_ *http.Request, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("new request: %w", err)
		}
	}()
	request, err := http.NewRequestWithContext(ctx, method, requestPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", getUserAgent())
	return request, nil
}

func getUserAgent() string {
	userAgent := "WayPlatformABAXGo"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		userAgent += "/" + info.Main.Version
	}
	return userAgent
}
