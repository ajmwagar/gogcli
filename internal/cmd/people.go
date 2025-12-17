package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func newPeopleCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "people",
		Short: "Google People",
	}
	cmd.AddCommand(newPeopleMeCmd(flags))
	return cmd
}

func newPeopleMeCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show your profile (people/me)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			svc, err := newPeopleContactsService(cmd.Context(), account)
			if err != nil {
				return err
			}

			person, err := svc.People.Get("people/me").
				PersonFields("names,emailAddresses,photos").
				Do()
			if err != nil {
				return err
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"person": person})
			}

			name := ""
			email := ""
			photo := ""
			if len(person.Names) > 0 && person.Names[0] != nil {
				name = person.Names[0].DisplayName
			}
			if len(person.EmailAddresses) > 0 && person.EmailAddresses[0] != nil {
				email = person.EmailAddresses[0].Value
			}
			if len(person.Photos) > 0 && person.Photos[0] != nil {
				photo = person.Photos[0].Url
			}

			if name != "" {
				u.Out().Printf("name\t%s", name)
			}
			if email != "" {
				u.Out().Printf("email\t%s", email)
			}
			if photo != "" {
				u.Out().Printf("photo\t%s", photo)
			}
			return nil
		},
	}
}
