package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	abax "github.com/way-platform/abax-go"
	"golang.org/x/oauth2"
)

// NewClient creates a new Abax API client using the current CLI credentials.
func NewClient(ctx context.Context) (*abax.Client, error) {
	cf, err := ReadFile()
	if err != nil {
		return nil, err
	}
	return abax.NewClient(
		ctx,
		abax.WithClientID(cf.ClientID),
		abax.WithClientSecret(cf.ClientSecret),
		abax.WithRefreshToken(cf.Token.RefreshToken),
	)
}

// NewCommand returns a new [cobra.Command] for CLI authentication.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Authenticate to the ABAX Open API",
		GroupID: "auth",
	}
	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(newLogoutCommand())
	return cmd
}

func newLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the ABAX Open API using interactive browser flow",
	}
	clientID := cmd.Flags().String("client-id", "", "client ID to use for authentication")
	clientSecret := cmd.Flags().String("client-secret", "", "client secret to use for authentication")
	sandbox := cmd.Flags().Bool("sandbox", false, "use sandbox environment")
	callbackURL := cmd.Flags().String("callback-url", "http://localhost:4000/abax/callback", "callback URL for OAuth redirect")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *clientID == "" {
			return fmt.Errorf("client-id is required")
		}
		if *clientSecret == "" {
			return fmt.Errorf("client-secret is required")
		}

		return performInteractiveLogin(cmd.Context(), *clientID, *clientSecret, *sandbox, *callbackURL, cmd)
	}
	return cmd
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout from the ABAX Open API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := removeFile(); err != nil {
				return err
			}
			cmd.Println("Logged out.")
			return nil
		},
	}
}

// performInteractiveLogin handles the full 3-legged OAuth flow with OIDC discovery
func performInteractiveLogin(
	ctx context.Context,
	clientID, clientSecret string,
	sandbox bool,
	callbackURL string,
	cmd *cobra.Command,
) error {
	// Discover OIDC endpoints
	issuer := "https://identity.abax.cloud"
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return fmt.Errorf("failed to discover OIDC endpoints: %w", err)
	}

	// Generate PKCE parameters
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Configure OAuth2 with discovered endpoints
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  callbackURL,
		Scopes:       getScopes(sandbox),
	}

	// Generate state parameter for security
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Parse callback URL to extract server address and path
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("invalid callback URL: %w", err)
	}

	// Start callback server
	callbackChan := make(chan callbackResult, 1)
	server := &http.Server{Addr: parsedURL.Host}

	http.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
		handleCallback(w, r, state, callbackChan)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			callbackChan <- callbackResult{Error: fmt.Errorf("server error: %w", err)}
		}
	}()

	// Build authorization URL with PKCE
	authURL := oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	cmd.Printf("Opening browser for authentication...\n")
	cmd.Printf("If the browser doesn't open automatically, please visit:\n%s\n\n", authURL)

	// Try to open browser automatically
	if err := maybeOpenBrowser(authURL); err != nil {
		cmd.Printf("Could not open browser automatically: %v\n", err)
	}

	cmd.Printf("Waiting for callback on %s...\n", callbackURL)

	// Wait for callback
	var result callbackResult
	select {
	case result = <-callbackChan:
	case <-ctx.Done():
		result.Error = ctx.Err()
	}

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	if result.Error != nil {
		return result.Error
	}

	// Exchange authorization code for tokens
	token, err := oauth2Config.Exchange(ctx, result.Code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save credentials
	if err := writeFile(&File{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Token:        *token,
	}); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	cmd.Printf("Successfully logged in!\n")
	return nil
}

type callbackResult struct {
	Code  string
	Error error
}

func handleCallback(
	w http.ResponseWriter,
	r *http.Request,
	expectedState string,
	callbackChan chan<- callbackResult,
) {
	// Check for errors in the callback
	if errorParam := r.URL.Query().Get("error"); errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		err := fmt.Errorf("authorization error: %s", errorParam)
		if errorDesc != "" {
			err = fmt.Errorf("authorization error: %s - %s", errorParam, errorDesc)
		}

		w.WriteHeader(http.StatusBadRequest)
		writeHTMLError(w, err)

		callbackChan <- callbackResult{Error: err}
		return
	}

	// Verify state parameter
	state := r.URL.Query().Get("state")
	if state != expectedState {
		err := fmt.Errorf("invalid state parameter")

		w.WriteHeader(http.StatusBadRequest)
		writeHTMLError(w, err)

		callbackChan <- callbackResult{Error: err}
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		err := fmt.Errorf("no authorization code received")

		w.WriteHeader(http.StatusBadRequest)
		writeHTMLError(w, err)

		callbackChan <- callbackResult{Error: err}
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "<html><body><h1>Authorization Successful</h1><p>You can close this window and return to the CLI.</p></body></html>")

	callbackChan <- callbackResult{Code: code}
}

func writeHTMLError(w http.ResponseWriter, err error) {
	_, _ = fmt.Fprintf(w, "<html><body><h1>Authorization Failed</h1><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(err.Error()))
}

func generateCodeVerifier() (string, error) {
	// Generate 43-128 character code verifier as per RFC 7636
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func generateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateState() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func getScopes(sandbox bool) []string {
	if sandbox {
		return []string{"openid", "abax_profile", "open_api.sandbox", "open_api.sandbox.vehicles", "offline_access"}
	}
	return []string{"openid", "abax_profile", "open_api", "open_api.vehicles", "offline_access"}
}

func maybeOpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return nil // unsupported platform, don't open the browser
	}
	return cmd.Start()
}

// File storing authentication credentials for the CLI.
type File struct {
	// TODO: Don't cache these sensitive fields on disk.
	ClientID     string       `json:"client_id"`
	ClientSecret string       `json:"client_secret"`
	Token        oauth2.Token `json:"token"`
}

func (cf *File) isExpired() bool {
	return cf.Token.Expiry.Before(time.Now())
}

func resolveFilepath() (string, error) {
	return xdg.ConfigFile("abax-go/auth.json")
}

// ReadFile reads the currently stored [File].
func ReadFile() (*File, error) {
	fp, err := resolveFilepath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fp); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no credentials found, please login using `abax auth login`")
		}
		return nil, err
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.isExpired() {
		return nil, fmt.Errorf("credentials expired, please login again")
	}
	return &f, nil
}

// writeFile writes the stored [credentialsFile].
func writeFile(f *File) error {
	fp, err := resolveFilepath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0o600)
}

// removeFile removes the stored [File].
func removeFile() error {
	fp, err := resolveFilepath()
	if err != nil {
		return err
	}
	return os.RemoveAll(fp)
}
