package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	abax "github.com/way-platform/abax-go"
	"golang.org/x/oauth2"
	"golang.org/x/term"
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
		Short: "Login to the ABAX Open API",
	}
	clientID := cmd.Flags().String("client-id", "-", "client ID to use for authentication")
	clientSecret := cmd.Flags().String("client-secret", "-", "client secret to use for authentication")
	refreshToken := cmd.Flags().String("refresh-token", "-", "refresh token to use for authentication")
	// TODO: Support 3-legged auth flow.
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *clientID == "" {
			cmd.Println("\nEnter client ID:")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			*clientID = string(input)
		}
		if *clientSecret == "-" {
			cmd.Println("\nEnter client secret:")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			*clientSecret = string(input)
		}
		if *refreshToken == "-" {
			cmd.Println("\nEnter refresh token:")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			*refreshToken = string(input)
		}
		oauth2Config := abax.NewOAuth2Config(*clientID, *clientSecret, *refreshToken)
		token, err := oauth2Config.Token(cmd.Context())
		if err != nil {
			return err
		}
		if err := writeFile(&File{
			ClientID:     *clientID,
			ClientSecret: *clientSecret,
			Token:        *token,
		}); err != nil {
			return err
		}
		cmd.Printf("Logged in.\n")
		return nil
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
