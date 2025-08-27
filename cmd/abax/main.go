package main

import (
	"context"
	"fmt"
	"image/color"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/way-platform/abax-go"
	"github.com/way-platform/abax-go/cmd/abax/internal/auth"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		newRootCommand(),
		fang.WithColorSchemeFunc(func(c lipgloss.LightDarkFunc) fang.ColorScheme {
			base := c(lipgloss.Black, lipgloss.White)
			baseInverted := c(lipgloss.White, lipgloss.Black)
			return fang.ColorScheme{
				Base:         base,
				Title:        base,
				Description:  base,
				Comment:      base,
				Flag:         base,
				FlagDefault:  base,
				Command:      base,
				QuotedString: base,
				Argument:     base,
				Help:         base,
				Dash:         base,
				ErrorHeader:  [2]color.Color{baseInverted, base},
				ErrorDetails: base,
			}
		}),
	); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "abax",
		Short: "ABAX Open API CLI",
	}
	cmd.AddGroup(auth.NewGroup())
	cmd.AddCommand(auth.NewCommand())
	cmd.AddGroup(&cobra.Group{
		ID:    "utils",
		Title: "Utils",
	})
	cmd.SetHelpCommandGroupID("utils")
	cmd.SetCompletionCommandGroupID("utils")
	return cmd
}

func newVehiclesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vehicles",
		Short:   "Vehicles",
		GroupID: "vehicles",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		client, err := abax.NewClient(cmd.Context())
		if err != nil {
			return err
		}
		request := &abax.ListVehiclesRequest{}
		for {
			response, err := client.ListVehicles(cmd.Context(), request)
			if err != nil {
				return err
			}
			for _, vehicle := range response.Vehicles {
				fmt.Println(protojson.Format(vehicle))
			}
			if len(response.Vehicles) < int(response.PageSize) {
				break
			}
			request.Page++
		}
		return nil
	}
	return cmd
}
