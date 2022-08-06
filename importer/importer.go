package importer

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bdxio/cfp-to-trello/cfp"
	"github.com/bdxio/cfp-to-trello/geo"
	"github.com/bdxio/cfp-to-trello/trello"
)

func ImportCFP(orgName, eventID, jsonPath string, locate geo.Locator, client trello.Client) error {
	event, err := cfp.Parse(jsonPath, locate)
	if err != nil {
		return err
	}
	t := Trello{eventID: eventID, client: client, event: event}
	return t.importCFP(orgName)
}

type Trello struct {
	eventID string
	client  trello.Client
	event   cfp.Event
}

func (t Trello) importCFP(organizationName string) error {
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		log.Printf("Successfully imported event %s in Trello in %v", t.event.Name, duration)
	}()

	organization, err := t.client.GetOrganization(organizationName)
	if err != nil {
		return err
	}

	log.Printf("Importing event %s in Trello organization %s...\n", t.event.Name, organizationName)
	g, _ := errgroup.WithContext(context.Background())
	for _, format := range t.event.Formats {
		g.Go(func(f string) func() error {
			return func() error {
				return t.createBoard(organization, f)
			}
		}(format))
	}
	return g.Wait()
}

func (t Trello) createBoard(organization trello.Organization, format string) error {
	proposals := t.event.GetProposals(format)
	if len(proposals) == 0 {
		return nil
	}

	// Create board
	boardName := fmt.Sprintf("Délibération %s - %s", t.event.Name, format)
	log.Printf("Creating board %s for %d proposals...\n", boardName, len(proposals))
	board, err := t.client.CreateBoard(organization, boardName, trello.PermissionLevelOrg)
	if err != nil {
		return err
	}

	// Create lists
	for _, name := range []string{trello.ListSelection, "Désistements", "Backups Acceptés", "Backups"} {
		_, err := t.client.CreateList(name, board)
		if err != nil {
			return err
		}
	}

	if err := t.createDeliberationLists(board, format); err != nil {
		return err
	}

	if _, err := t.client.CreateList("Refusés", board); err != nil {
		return err
	}

	log.Printf("Successfully created board %s: %s", board.Name, board.URL)
	return nil
}

func (t Trello) createDeliberationLists(board trello.Board, format string) error {
	proposalsByCategory := t.event.GetProposalsByCategory(format)
	lastTierProposals := make([]cfp.Proposal, 0)
	for _, category := range t.event.Categories {
		proposals := proposalsByCategory[category]
		if len(proposals) == 0 {
			continue
		}
		remainingProposals, err := t.createCategoryDeliberationLists(board, category, proposals)
		if err != nil {
			return err
		}
		if len(remainingProposals) > 0 {
			lastTierProposals = append(lastTierProposals, remainingProposals...)
		}
	}
	return t.createDeliberationList(board, "T3", lastTierProposals)
}

func (t Trello) createCategoryDeliberationLists(board trello.Board, category string, proposals []cfp.Proposal) ([]cfp.Proposal, error) {
	sort.Slice(proposals, func(i, j int) bool {
		p1 := proposals[i]
		p2 := proposals[j]
		if p1.Rating > p2.Rating {
			return true
		}
		if p1.Rating < p2.Rating {
			return false
		}
		// p1 and p2 have equal ratings
		if p1.Loves > p2.Loves {
			return true
		}
		if p1.Loves < p2.Loves {
			return false
		}
		// p1 and p2 received the same love, only hate can split the tie
		if p1.Hates < p2.Hates {
			return true
		}
		return false
	})

	size := int(math.Ceil(float64(len(proposals)) / 3))
	if err := t.createDeliberationList(board, fmt.Sprintf("%s - T1", category), proposals[:size]); err != nil {
		return nil, err
	}
	if err := t.createDeliberationList(board, fmt.Sprintf("%s - T2", category), proposals[size:size*2]); err != nil {
		return nil, err
	}
	return proposals[size*2:], nil
}

func (t Trello) createDeliberationList(board trello.Board, name string, proposals []cfp.Proposal) error {
	log.Printf("Creating deliberation list %s for %d proposals...\n", name, len(proposals))
	list, err := t.client.CreateList(name, board)
	if err != nil {
		return err
	}
	for _, proposal := range proposals {
		if err := t.createProposalCard(board, list, proposal); err != nil {
			return err
		}
	}
	return nil
}

func (t Trello) createProposalCard(board trello.Board, list trello.List, proposal cfp.Proposal) error {
	log.Printf("Creating proposal card %s...\n", proposal.Title)
	labels, err := t.createLabels(board, proposal,
		func(p cfp.Proposal) (string, trello.Color) { return p.Category, trello.ColorGreen },
		func(p cfp.Proposal) (string, trello.Color) {
			return fmt.Sprintf("🏅 %1.1f", p.Rating), trello.ColorOrange
		},
		func(p cfp.Proposal) (string, trello.Color) {
			return fmt.Sprintf("%d ❤️ / %d ☠️", p.Loves, p.Hates), trello.ColorRed
		},
		func(p cfp.Proposal) (string, trello.Color) { return p.Speakers, trello.ColorPurple },
		func(p cfp.Proposal) (string, trello.Color) { return p.AudienceLevel, trello.ColorSky },
		func(p cfp.Proposal) (string, trello.Color) { return p.Language, trello.ColorPink },
	)
	if err != nil {
		return err
	}

	proposalUrl := fmt.Sprintf("%s/organizer/event/%s/proposals/%s", cfp.URL, t.eventID, proposal.ID)
	proposalLink := fmt.Sprintf("📜 [Proposal](%s)", proposalUrl)
	cardDescription := fmt.Sprintf("%s\n\n---\n\n%s\n\n---\n\n%s", proposalLink, proposal.Abstract, proposal.PrivateMessage)
	card, err := t.client.CreateCard(proposal.Title, cardDescription, list, labels)
	if err != nil {
		return err
	}

	for _, message := range proposal.OrganizerMessages {
		if err := t.client.CreateComment(message, card); err != nil {
			return err
		}
	}
	return nil
}

func (t Trello) createLabels(board trello.Board, p cfp.Proposal, markers ...func(p cfp.Proposal) (string, trello.Color)) ([]trello.Label, error) {
	labels := make([]trello.Label, 0)
	for _, marker := range markers {
		name, color := marker(p)
		label, err := t.client.CreateLabel(name, board, color)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, nil
}
