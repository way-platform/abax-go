package auth

import (
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
func NewClient() (*abax.Client, error) {
	cf, err := readFile()
	if err != nil {
		return nil, err
	}
	_ = cf // TODO: Provide auth.
	return abax.NewClient(
	// TODO: Provide auth.
	)
}

// NewCommand returns a new [cobra.Command] for CLI authentication.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Authenticate to the Abax API",
		GroupID: "auth",
	}
	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(newLogoutCommand())
	return cmd
}

func newLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Abax API",
	}
	token := cmd.Flags().String("token", "-", "access token to use for authentication")
	// TODO: Support 3-legged auth flow.
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *token == "-" {
			cmd.Println("\nEnter access token:")
			input, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			*token = string(input)
		}
		if err := writeFile(&File{
			Token: oauth2.Token{
				AccessToken: *token,
			},
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
		Short: "Logout from the Abax API",
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
	Token oauth2.Token `json:"token"`
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
