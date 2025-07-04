package importer

import (
	"context"
	"fmt"
	"html"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bdxio/cfp-to-trello/cfp"
	"github.com/bdxio/cfp-to-trello/geo"
	"github.com/bdxio/cfp-to-trello/trello"
)

// Maximum size limits for Trello API to avoid 413 errors
const (
	maxDescriptionLength = 8000  // More aggressive limit to avoid 413 errors
	maxCommentLength     = 8000  // More aggressive limit to avoid 413 errors
	maxLabelsPerCard     = 3     // Limit the number of labels to reduce payload size
)

// escapeHTML escapes any HTML special characters like < > & in a string
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// escapeCFPProposal returns a copy of the proposal with all text fields HTML-escaped
func escapeCFPProposal(p cfp.Proposal) cfp.Proposal {
	escaped := p
	escaped.Title = escapeHTML(p.Title)
	escaped.Abstract = escapeHTML(p.Abstract)
	escaped.PrivateMessage = escapeHTML(p.PrivateMessage)
	escaped.Category = escapeHTML(p.Category)
	escaped.Speakers = escapeHTML(p.Speakers)
	escaped.AudienceLevel = escapeHTML(p.AudienceLevel)
	escaped.Language = escapeHTML(p.Language)

	// Escape organizer messages
	escapedMessages := make([]string, len(p.OrganizerMessages))
	for i, msg := range p.OrganizerMessages {
		escapedMessages[i] = escapeHTML(msg)
	}
	escaped.OrganizerMessages = escapedMessages

	return escaped
}

// truncateDescription ensures the card description stays within Trello's size limit
func truncateDescription(description string) string {
	if len(description) <= maxDescriptionLength {
		return description
	}

	// Keep the proposal link and some part of the abstract
	parts := []string{"", "", ""}
	sections := []string{}

	// Try to find the separator "---" to split the parts
	currentPart := 0
	for _, line := range strings.Split(description, "\n") {
		if line == "---" {
			currentPart++
			if currentPart > 2 {
				break
			}
			continue
		}

		if currentPart < len(parts) {
			parts[currentPart] += line + "\n"
		}
	}

	// Always keep the first part (proposal link)
	sections = append(sections, strings.TrimSpace(parts[0]))

	// Calculate remaining space
	remainingSpace := maxDescriptionLength - len(sections[0]) - 20 // 20 chars for separators and buffer

	// Try to include some of the abstract (part 1)
	if len(parts[1]) > 0 {
		abstractPart := parts[1]
		if len(abstractPart) > remainingSpace/2 {
			abstractPart = abstractPart[:remainingSpace/2] + "...(truncated)"
		}
		sections = append(sections, strings.TrimSpace(abstractPart))
		remainingSpace -= len(abstractPart)
	}

	// Try to include some of the private message (part 2)
	if len(parts[2]) > 0 && remainingSpace > 30 {
		privatePart := parts[2]
		if len(privatePart) > remainingSpace {
			privatePart = privatePart[:remainingSpace] + "...(truncated)"
		}
		sections = append(sections, strings.TrimSpace(privatePart))
	}

	// Put it all back together with separators
	return strings.Join(sections, "\n\n---\n\n")
}

// truncateComment ensures a comment stays within Trello's size limit
func truncateComment(comment string) string {
	if len(comment) <= maxCommentLength {
		return comment
	}
	return comment[:maxCommentLength-12] + "...(truncated)"
}

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
	// Sort the proposals by rating, loves, and hates
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

	// Calculate size of each tier
	totalProposals := len(proposals)
	
	// Handle case with very few proposals
	if totalProposals <= 1 {
		// If we have 0 or 1 proposal, just put it in T1 and return empty for T3
		if totalProposals == 1 {
			if err := t.createDeliberationList(board, fmt.Sprintf("%s - T1", category), proposals); err != nil {
				return nil, err
			}
		}
		// Return empty slice for T3
		return []cfp.Proposal{}, nil
	}
	
	// Handle case with 2 proposals
	if totalProposals == 2 {
		// Put one in T1 and one in T2
		if err := t.createDeliberationList(board, fmt.Sprintf("%s - T1", category), proposals[:1]); err != nil {
			return nil, err
		}
		if err := t.createDeliberationList(board, fmt.Sprintf("%s - T2", category), proposals[1:2]); err != nil {
			return nil, err
		}
		// Return empty slice for T3
		return []cfp.Proposal{}, nil
	}
	
	// Normal case with 3 or more proposals
	size := int(math.Ceil(float64(totalProposals) / 3))
	
	// T1 gets the top tier
	if err := t.createDeliberationList(board, fmt.Sprintf("%s - T1", category), proposals[:size]); err != nil {
		return nil, err
	}
	
	// Make sure we don't go out of bounds for T2
	t2End := size * 2
	if t2End > totalProposals {
		t2End = totalProposals
	}
	
	// Only create T2 if there are proposals for it
	if size < t2End {
		if err := t.createDeliberationList(board, fmt.Sprintf("%s - T2", category), proposals[size:t2End]); err != nil {
			return nil, err
		}
	}
	
	// Return remaining proposals for T3, if any
	if t2End < totalProposals {
		return proposals[t2End:], nil
	}
	
	// No proposals for T3
	return []cfp.Proposal{}, nil
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
	// Escape HTML characters in the proposal to prevent JSON parsing issues
	escapedProposal := escapeCFPProposal(proposal)

	log.Printf("Creating proposal card %s...\n", escapedProposal.Title)
	
	// Define all possible labels we might want to use
	labelFunctions := []func(p cfp.Proposal) (string, trello.Color){
		// Most important labels first
		func(p cfp.Proposal) (string, trello.Color) { return p.Category, trello.ColorGreen },
		func(p cfp.Proposal) (string, trello.Color) { 
			return fmt.Sprintf("🏅 %1.1f", p.Rating), trello.ColorOrange 
		},
		func(p cfp.Proposal) (string, trello.Color) { 
			return fmt.Sprintf("%d ❤️ / %d ☠️", p.Loves, p.Hates), trello.ColorRed 
		},
		// Less important labels last (these might be skipped)
		func(p cfp.Proposal) (string, trello.Color) { return p.AudienceLevel, trello.ColorSky },
		func(p cfp.Proposal) (string, trello.Color) { return p.Language, trello.ColorPink },
		func(p cfp.Proposal) (string, trello.Color) { return p.Speakers, trello.ColorPurple },
	}
	
	// Only use the first maxLabelsPerCard labels to reduce payload size
	// Using if statement instead of min() function for Go 1.19 compatibility
	labelCount := len(labelFunctions)
	if labelCount > maxLabelsPerCard {
		labelCount = maxLabelsPerCard
	}
	
	labels, err := t.createLabels(board, escapedProposal, labelFunctions[:labelCount]...)
	if err != nil {
		return err
	}

	// Create a more compact description to reduce payload size
	proposalUrl := fmt.Sprintf("%s/organizer/event/%s/proposals/%s", cfp.URL, t.eventID, escapedProposal.ID)
	proposalLink := fmt.Sprintf("📜 [Proposal](%s)", proposalUrl)
	
	// Include key metadata in the description to make up for fewer labels
	metadataText := fmt.Sprintf("Category: %s | Rating: %1.1f | Level: %s | Lang: %s", 
		escapedProposal.Category, 
		escapedProposal.Rating,
		escapedProposal.AudienceLevel,
		escapedProposal.Language)
	
	cardDescription := fmt.Sprintf("%s\n\n%s\n\n---\n\n%s", 
		proposalLink, 
		metadataText,
		escapedProposal.Abstract)
	
	// Only include private message if it's not empty
	if escapedProposal.PrivateMessage != "" {
		cardDescription += fmt.Sprintf("\n\n---\n\n%s", escapedProposal.PrivateMessage)
	}
	
	cardDescription = truncateDescription(cardDescription)
	card, err := t.client.CreateCard(escapedProposal.Title, cardDescription, list, labels)
	if err != nil {
		return err
	}

	// Limit the number of comments to reduce API calls
	for i, message := range escapedProposal.OrganizerMessages {
		// Skip empty messages
		if len(strings.TrimSpace(message)) == 0 {
			continue
		}
		
		truncatedMessage := truncateComment(message)
		if err := t.client.CreateComment(truncatedMessage, card); err != nil {
			return err
		}
		
		// Add a small delay between comment creation to prevent rate limiting
		if i < len(escapedProposal.OrganizerMessages)-1 {
			time.Sleep(100 * time.Millisecond)
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
