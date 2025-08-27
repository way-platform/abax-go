package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	abax "github.com/way-platform/abax-go"
	"golang.org/x/oauth2"
	"golang.org/x/term"
)

// NewClient creates a new Abax API client using the current CLI credentials.
func NewClient(ctx context.Context) (*abax.Client, error) {
	cf, err := readFile()
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
		Short: "Login to the ABAX Open API",
	}
	clientID := cmd.Flags().String("client-id", "", "client ID to use for authentication")
	clientSecret := cmd.Flags().String("client-secret", "", "client secret to use for authentication")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *clientID == "" {
			cmd.Print("Enter client ID: ")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			cmd.Println() // Print newline after password input
			*clientID = string(input)
		}
		if *clientSecret == "" {
			cmd.Print("Enter client secret: ")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			cmd.Println() // Print newline after password input
			*clientSecret = string(input)
		}

		// Use interactive authorization code flow
		oauth2Config, err := newAuthCodeOAuth2Config(*clientID, *clientSecret)
		if err != nil {
			return fmt.Errorf("creating OAuth2 config: %w", err)
		}

		token, err := getTokenInteractive(cmd, oauth2Config)
		if err != nil {
			return fmt.Errorf("getting token via interactive login: %w", err)
		}

		if err := writeFile(&File{
			ClientID:     *clientID,
			ClientSecret: *clientSecret,
			Token:        *token,
		}); err != nil {
			return err
		}
		cmd.Printf("Logged in successfully.\n")
		return nil
	}
	return cmd
}

// newAuthCodeOAuth2Config creates an OAuth2 config for authorization code flow using OIDC discovery
func newAuthCodeOAuth2Config(clientID, clientSecret string) (*oauth2.Config, error) {
	// Use OIDC discovery like the working backend
	provider, err := oidc.NewProvider(context.Background(), "https://identity.abax.cloud")
	if err != nil {
		return nil, fmt.Errorf("creating OIDC provider: %w", err)
	}

	// Create the config exactly like your working backend
	oauth2Config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:3000/abax/callback",
		// Discovery returns the OAuth2 endpoints
		Endpoint: provider.Endpoint(),
		// "openid" is a required scope for OpenID Connect flows
		Scopes: []string{oidc.ScopeOpenID, "abax_profile", "open_api", "open_api.vehicles", "offline_access"},
	}

	return &oauth2Config, nil
}

func getTokenInteractive(cmd *cobra.Command, oauth2Config *oauth2.Config) (*oauth2.Token, error) {
	state, err := newState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start HTTP server on localhost:3000 to handle the callback
	mux := http.NewServeMux()
	mux.HandleFunc("/abax/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state parameter")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errorParam := r.URL.Query().Get("error")
			errorDesc := r.URL.Query().Get("error_description")
			if errorParam != "" {
				http.Error(w, fmt.Sprintf("Authorization error: %s - %s", errorParam, errorDesc), http.StatusBadRequest)
				errChan <- fmt.Errorf("authorization error: %s - %s", errorParam, errorDesc)
				return
			}
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			errChan <- fmt.Errorf("authorization code not found in callback")
			return
		}

		// Success page
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `
<!DOCTYPE html>
<html>
<head>
    <title>ABAX CLI - Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 40px; text-align: center; }
        .success { color: #28a745; }
        .container { max-width: 600px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="success">✓ Authentication Successful!</h1>
        <p>You have successfully authenticated with the ABAX Open API.</p>
        <p>You can now close this window and return to your terminal.</p>
    </div>
</body>
</html>`)

		codeChan <- code
	})

	server := &http.Server{
		Addr:    ":3000",
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("starting callback server: %w", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	cmd.Printf("Opening browser for authentication...\n")
	cmd.Printf("If the browser doesn't open automatically, please visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		cmd.Printf("Failed to open browser automatically: %v\n", err)
		cmd.Printf("Please visit the URL above manually.\n")
	}

	cmd.Printf("Waiting for authentication callback...\n")

	select {
	case code := <-codeChan:
		cmd.Printf("Received authorization code, exchanging for tokens...\n")
		return oauth2Config.Exchange(cmd.Context(), code)
	case err := <-errChan:
		return nil, err
	case <-cmd.Context().Done():
		return nil, cmd.Context().Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout after 5 minutes")
	}
}

func newState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Use RawURLEncoding to avoid problematic characters like + and =
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser attempts to open the given URL in the user's default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
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

// readFile reads the currently stored [File].
func readFile() (*File, error) {
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
