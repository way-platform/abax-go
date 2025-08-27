package abax

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"golang.org/x/oauth2"
)

// Client to the Abax management APIs.
type Client struct {
	config     ClientConfig
	httpClient *http.Client
}

// NewClient creates a new Abax API client.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	client := Client{
		config: newClientConfig(),
	}
	for _, opt := range opts {
		opt(&client.config)
	}
	if client.config.clientID == "" || client.config.clientSecret == "" {
		return nil, fmt.Errorf("invalid client config: must provide clientID and clientSecret")
	}
	oauth2Config := NewOAuth2Config(client.config.clientID, client.config.clientSecret)

	// If we have a refresh token, use it to get a fresh access token
	if client.config.refreshToken != "" {
		token := &oauth2.Token{RefreshToken: client.config.refreshToken}
		client.httpClient = oauth2Config.Client(ctx, token)
	} else if client.config.accessToken != "" {
		// Use the provided access token directly
		token := &oauth2.Token{AccessToken: client.config.accessToken}
		client.httpClient = oauth2Config.Client(ctx, token)
	} else {
		return nil, fmt.Errorf("invalid client config: must provide either refreshToken or accessToken")
	}
	return &client, nil
}

// NewOAuth2Config creates a new OAuth2 config for the Abax API using authorization code flow.
func NewOAuth2Config(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://identity.abax.cloud/connect/authorize",
			TokenURL: "https://identity.abax.cloud/connect/token",
		},
		RedirectURL: "http://localhost:4000/abax/callback", // Default callback URL
		Scopes:      []string{"openid", "abax_profile", "open_api", "open_api.vehicles", "offline_access"},
	}
}

// ClientConfig configures a [Client].
type ClientConfig struct {
	baseURL      string
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
}

func newClientConfig() ClientConfig {
	return ClientConfig{
		baseURL: "https://api.abax.cloud",
	}
}

// ClientOption is a configuration option for a [Client].
type ClientOption func(*ClientConfig)

// WithClientID sets the client ID for the client.
func WithClientID(clientID string) ClientOption {
	return func(config *ClientConfig) {
		config.clientID = clientID
	}
}

// WithClientSecret sets the client secret for the client.
func WithClientSecret(clientSecret string) ClientOption {
	return func(config *ClientConfig) {
		config.clientSecret = clientSecret
	}
}

// WithRefreshToken sets the refresh token for the client.
func WithRefreshToken(refreshToken string) ClientOption {
	return func(config *ClientConfig) {
		config.refreshToken = refreshToken
	}
}

// WithAccessToken sets the access token for the client.
func WithAccessToken(accessToken string) ClientOption {
	return func(config *ClientConfig) {
		config.accessToken = accessToken
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	method string,
	requestPath string,
	body io.Reader,
) (_ *http.Request, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("new request: %w", err)
		}
	}()
	request, err := http.NewRequestWithContext(ctx, method, c.config.baseURL+requestPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", getUserAgent())
	return request, nil
}

// Error is an error returned by the Abax API.
type Error struct {
	// Status is the HTTP status code of the response.
	Status string `json:"code"`
	// TODO: Parse more error fields.
}

func (e *Error) Error() string {
	return fmt.Sprintf("response error: %s", e.Status)
}

func (c *Client) newResponseError(response *http.Response) error {
	// TODO: Parse any error message from the body.
	return &Error{
		Status: response.Status,
	}
}

func getUserAgent() string {
	userAgent := "WayPlatformABAXGo"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		userAgent += "/" + info.Main.Version
	}
	return userAgent
}
