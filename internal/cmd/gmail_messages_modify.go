package cmd

import (
	"context"
	"os"

	"google.golang.org/api/gmail/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type GmailMessagesModifyCmd struct {
	IDs         []string `name:"ids" required:"" help:"Message IDs (comma-separated or repeated)"`
	AddLabel    string   `name:"add-label" help:"Labels to add (comma-separated, name or ID)"`
	RemoveLabel string   `name:"remove-label" help:"Labels to remove (comma-separated, name or ID)"`
	Archive     bool     `name:"archive" help:"Archive messages (remove from INBOX)"`
}

func (c *GmailMessagesModifyCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	if len(c.IDs) == 0 {
		return usage("must specify --ids")
	}

	addLabels := splitCSV(c.AddLabel)
	removeLabels := splitCSV(c.RemoveLabel)

	if len(addLabels) == 0 && len(removeLabels) == 0 && !c.Archive {
		return usage("must specify --add-label, --remove-label, and/or --archive")
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	idMap, err := fetchLabelNameToID(svc)
	if err != nil {
		return err
	}

	addIDs := resolveLabelIDs(addLabels, idMap)
	removeIDs := resolveLabelIDs(removeLabels, idMap)

	// Archive means remove from INBOX
	if c.Archive {
		// Check if INBOX is already in removeIDs to avoid duplicates
		hasInbox := false
		for _, id := range removeIDs {
			if id == "INBOX" {
				hasInbox = true
				break
			}
		}
		if !hasInbox {
			removeIDs = append(removeIDs, "INBOX")
		}
	}

	err = svc.Users.Messages.BatchModify("me", &gmail.BatchModifyMessagesRequest{
		Ids:            c.IDs,
		AddLabelIds:    addIDs,
		RemoveLabelIds: removeIDs,
	}).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"modified":      c.IDs,
			"count":         len(c.IDs),
			"addedLabels":   addIDs,
			"removedLabels": removeIDs,
			"archived":      c.Archive,
		})
	}

	u.Out().Printf("Modified %d message(s)", len(c.IDs))
	return nil
}
