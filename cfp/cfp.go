package cfp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bdxio/cfp-to-trello/geo"
)

const stateSubmitted = "submitted"

type Event struct {
	Name       string
	Proposals  []Proposal
	Formats    []string
	Categories []string
}

func (e Event) GetProposals(format string) []Proposal {
	p := make([]Proposal, 0)
	for _, proposal := range e.Proposals {
		if proposal.Format == format {
			p = append(p, proposal)
		}
	}
	return p
}

func (e Event) GetProposalsByCategory(format string) map[string][]Proposal {
	proposals := e.GetProposals(format)
	m := make(map[string][]Proposal)
	for _, proposal := range proposals {
		_, ok := m[proposal.Category]
		if !ok {
			m[proposal.Category] = make([]Proposal, 0)
		}
		m[proposal.Category] = append(m[proposal.Category], proposal)
	}
	return m
}

type Proposal struct {
	ID                string
	Title             string
	Category          string
	Format            string
	Abstract          string
	AudienceLevel     string
	Language          string
	Speakers          string
	PrivateMessage    string
	Rating            float64
	Loves             int
	Hates             int
	OrganizerMessages []string
}

type Export struct {
	Name       string     `json:"name"`
	Categories []Category `json:"categories"`
	Formats    []Format   `json:"formats"`
	Talks      []Talk     `json:"talks"`
	Speakers   []Speaker  `json:"speakers"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Format struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Talk struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	State            string            `json:"state"`
	Level            string            `json:"level"`
	Abstract         string            `json:"abstract"`
	Categories       string            `json:"categories"` // plural but holds only one category
	Formats          string            `json:"formats"`    // plural but holds only one format
	Speakers         []string          `json:"speakers"`
	Comments         string            `json:"comments"`
	Rating           float64           `json:"rating"`
	Loves            int               `json:"loves"`
	Hates            int               `json:"hates"`
	Language         string            `json:"language"`
	OrganizersThread []OrganizerThread `json:"organizersThread"`
}

func (t Talk) IsSubmitted() bool {
	return t.State == stateSubmitted
}

type OrganizerThread struct {
	DisplayName string `json:"displayName"`
	Message     string `json:"message"`
	Date        Date   `json:"date"`
}

type Date struct {
	Seconds     int64 `json:"_seconds"`
	Nanoseconds int64 `json:"_nanoseconds"`
}

type Speaker struct {
	UID         string   `json:"uid"`
	DisplayName string   `json:"displayName"`
	Company     string   `json:"company"`
	Address     *Address `json:"address"`
	Email       string   `json:"email"`
}

type Address struct {
	LatLng           LatLng `json:"latLng"`
	FormattedAddress string `json:"formattedAddress"`
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

var audienceLevels = map[string]string{
	"beginner":     "Débutant",
	"intermediate": "Intermédiaire",
	"advanced":     "Avancé",
}

var languages = map[string]string{
	"fr":                                   "🇫🇷", // ISO code for French
	"en":                                   "🇬🇧", // ISO code for English
	"es":                                   "🇪🇸", // ISO code for Spanish
}

// Parse checks the JSON format and chooses the appropriate parser
func Parse(path string, locate geo.Locator) (Event, error) {
	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return Event{}, err
	}

	// Pre-process the data to handle any invalid characters
	// Replace problematic HTML-like content (e.g., <tag>) with escaped versions
	cleanedData := cleanJSONData(data)

	// Check if it's an array (Conference Hall format) or an object (original format)
	previewSize := 100
	if len(cleanedData) < 100 {
		previewSize = len(cleanedData)
	}
	trimmedBytes := strings.TrimSpace(string(cleanedData[:previewSize]))
	if strings.HasPrefix(trimmedBytes, "[") {
		// Conference Hall format (array of objects)
		var proposals []ConferenceHallProposal
		if err := json.Unmarshal(cleanedData, &proposals); err != nil {
			return Event{}, fmt.Errorf("error unmarshaling Conference Hall format: %w", err)
		}
		return parseConferenceHallProposals(proposals, locate)
	} else if strings.HasPrefix(trimmedBytes, "{") {
		// Original format
		var export Export
		if err := json.Unmarshal(cleanedData, &export); err != nil {
			return Event{}, fmt.Errorf("error unmarshaling original format: %w", err)
		}
		return parseExport(export, path, locate)
	}

	return Event{}, fmt.Errorf("unknown JSON format in file: %s", path)
}

// cleanJSONData pre-processes JSON data to handle special characters and HTML-like content
func cleanJSONData(data []byte) []byte {
	// Convert to string for easier manipulation
	content := string(data)

	// If we detect that we need to continue even with invalid JSON,
	// we can implement various strategies like:

	// 1. Replace HTML tags with escaped versions
	// This is a simple approach that works for basic cases
	content = strings.ReplaceAll(content, "<", "\\u003c")
	content = strings.ReplaceAll(content, ">", "\\u003e")

	// 2. Handle other potentially problematic characters
	// Add more replacements if needed

	// Return the cleaned content as bytes
	return []byte(content)
}

func parseExport(export Export, path string, locate geo.Locator) (Event, error) {
	log.Printf("Parsing CFP export from %s...", path)
	categories := getCategories(export.Categories)
	formats := getFormats(export.Formats)
	speakers, err := getSpeakers(export.Speakers, locate)
	if err != nil {
		return Event{}, err
	}

	proposals := make([]Proposal, 0, len(export.Talks))
	for _, talk := range export.Talks {
		language, err := parseLanguage(talk.Language)
		if err != nil {
			return Event{}, err
		}
		speakerLabels := make([]string, 0, len(talk.Speakers))
		for _, speaker := range talk.Speakers {
			speakerLabel, ok := speakers[speaker]
			if !ok {
				return Event{}, fmt.Errorf("speaker %s not found in speakers map", speaker)
			}
			speakerLabels = append(speakerLabels, speakerLabel)
		}
		p := Proposal{
			ID:                talk.ID,
			Title:             strings.Trim(talk.Title, " "),
			Category:          categories[talk.Categories],
			Format:            formats[talk.Formats],
			Abstract:          talk.Abstract,
			AudienceLevel:     audienceLevels[talk.Level],
			Language:          language,
			Speakers:          strings.Join(speakerLabels, " / "),
			PrivateMessage:    talk.Comments,
			Rating:            talk.Rating,
			Loves:             talk.Loves,
			Hates:             talk.Hates,
			OrganizerMessages: parseOrganizerMessages(talk.OrganizersThread),
		}
		proposals = append(proposals, p)
	}

	return Event{Name: export.Name, Proposals: proposals, Formats: getValues(formats), Categories: getValues(categories)}, nil
}

func parseConferenceHallProposals(proposals []ConferenceHallProposal, locate geo.Locator) (Event, error) {
	log.Printf("Parsing Conference Hall export... Found %d proposals", len(proposals))

	// Extract formats
	formatMap := make(map[string]string)
	for _, proposal := range proposals {
		for _, format := range proposal.Formats {
			formatMap[format.ID] = format.Name
		}
	}

	// Extract categories
	categoryMap := make(map[string]string)
	for _, proposal := range proposals {
		for _, category := range proposal.Categories {
			categoryMap[category.ID] = category.Name
		}
	}

	// Convert to our internal Proposal format
	eventProposals := make([]Proposal, 0, len(proposals))
	for _, p := range proposals {
		// Get language
		var language string
		if len(p.Languages) > 0 {
			lang, err := parseLanguage(p.Languages[0])
			if err != nil {
				return Event{}, err
			}
			language = lang
		} else {
			// Default to French
			language = languages["French"]
		}

		// Get format and category
		var format, category string
		if len(p.Formats) > 0 {
			format = p.Formats[0].Name
		}
		if len(p.Categories) > 0 {
			category = p.Categories[0].Name
		}

		// Process speakers
		speakerLabels := make([]string, 0, len(p.Speakers))
		for _, speaker := range p.Speakers {
			speakerLabel := speaker.Name
			if speakerLabel == "" {
				speakerLabel = speaker.Email
			}
			speakerLabel += " -"

			// Process location if available
			if speaker.Location != "" {
				// Simple location handling without geo lookup
				speakerLabel += " " + speaker.Location

				// Check if in Gironde (simplified)
				if strings.Contains(strings.ToLower(speaker.Location), "bordeaux") ||
					strings.Contains(strings.ToLower(speaker.Location), "gironde") {
					speakerLabel += " 🍷"
				} else {
					speakerLabel += " 🗺️"
				}
			} 

			if speaker.Company != "" {
				speakerLabel += " (" + speaker.Company + ")"
			}

			speakerLabels = append(speakerLabels, speakerLabel)
		}

		// Create proposal
		proposal := Proposal{
			ID:            p.ID,
			Title:         strings.Trim(p.Title, " "),
			Category:      category,
			Format:        format,
			Abstract:      p.Abstract,
			AudienceLevel: audienceLevels[strings.ToLower(p.Level)],
			Language:      language,
			Speakers:      strings.Join(speakerLabels, " / "),
			PrivateMessage: p.References,
			Rating:        p.Review.Average,
			Loves:         p.Review.Positives,
			Hates:         p.Review.Negatives,
			OrganizerMessages: []string{}, // No organizer messages in this format
		}
		eventProposals = append(eventProposals, proposal)
	}

	// Get unique formats and categories
	formatValues := getValues(formatMap)
	categoryValues := getValues(categoryMap)

	return Event{
		Name:       "Conference Hall Import",
		Proposals:  eventProposals,
		Formats:    formatValues,
		Categories: categoryValues,
	}, nil
}

func getCategories(categories []Category) map[string]string {
	m := make(map[string]string)
	for _, category := range categories {
		m[category.ID] = category.Name
	}
	return m
}

func getFormats(formats []Format) map[string]string {
	m := make(map[string]string)
	for _, format := range formats {
		m[format.ID] = format.Name
	}
	return m
}

func getSpeakers(speakers []Speaker, locate geo.Locator) (map[string]string, error) {
	m := make(map[string]string)
	for _, speaker := range speakers {
		speakerLabel := speaker.DisplayName
		if speakerLabel == "" {
			speakerLabel = speaker.Email
		}
		speakerLabel += " -"

		if speaker.Address == nil {
			speakerLabel += " 🗺️"
		} else {
			location, err := locate(speaker.Address.LatLng.Lat, speaker.Address.LatLng.Lng, speaker.Address.FormattedAddress)
			if err != nil {
				return nil, err
			}
			speakerLabel += " " + location.City
			// Speaker in Gironde area should be identified clearly, a glass of wine should do the trick.
			if location.IsInGironde() {
				speakerLabel += " 🍷"
			}
		}

		if speaker.Company != "" {
			speakerLabel += " (" + speaker.Company + ")"
		}

		m[speaker.UID] = speakerLabel
	}
	return m, nil
}

func parseLanguage(language string) (string, error) {
	// French as default if empty language
	if language == "" || strings.Trim(language, " ") == "" {
		return languages["French"], nil
	}
	l, ok := languages[language]
	if !ok {
		return "", fmt.Errorf("%s is not a known language", language)
	}
	return l, nil
}

func parseOrganizerMessages(threads []OrganizerThread) []string {
	sort.Slice(threads, func(i, j int) bool {
		t1 := threads[i]
		t2 := threads[j]
		if t1.Date.Seconds < t2.Date.Seconds {
			return false
		}
		return true
	})

	msgs := make([]string, 0, len(threads))
	for _, thread := range threads {
		ts := time.Unix(thread.Date.Seconds, thread.Date.Nanoseconds)
		date := ts.Format("le 02/01 à 15h04")
		msgs = append(msgs, fmt.Sprintf("%s\n--\n**%s** _%s_", thread.Message, thread.DisplayName, date))
	}
	return msgs
}

func getValues(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	sort.Strings(s)
	return s
}

type ConferenceHallProposal struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Abstract    string                 `json:"abstract"`
	Level       string                 `json:"level"`
	References  string                 `json:"references"`
	Review      ConferenceHallReview   `json:"review"`
	Speakers    []ConferenceHallSpeaker `json:"speakers"`
	Categories  []Category             `json:"categories"`
	Formats     []Format               `json:"formats"`
	Languages   []string                `json:"languages"`
}

type ConferenceHallReview struct {
	Average   float64 `json:"average"`
	Positives int      `json:"positives"`
	Negatives int      `json:"negatives"`
}

type ConferenceHallSpeaker struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Location string `json:"location"`
}

// ParseOriginal parses the original JSON format with object structure
func ParseOriginal(path string, locate geo.Locator) (Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return Event{}, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return Event{}, err
	}

	var export Export
	if err := json.Unmarshal(data, &export); err != nil {
		return Event{}, err
	}

	log.Printf("Parsing CFP export from %s...", path)
	categories := getCategories(export.Categories)
	formats := getFormats(export.Formats)
	speakers, err := getSpeakers(export.Speakers, locate)
	if err != nil {
		return Event{}, err
	}

	proposals := make([]Proposal, 0, len(export.Talks))
	for _, talk := range export.Talks {
		language, err := parseLanguage(talk.Language)
		if err != nil {
			return Event{}, err
		}
		speakerLabels := make([]string, 0, len(talk.Speakers))
		for _, speaker := range talk.Speakers {
			speakerLabel, ok := speakers[speaker]
			if !ok {
				return Event{}, fmt.Errorf("speaker %s not found in speakers map", speaker)
			}
			speakerLabels = append(speakerLabels, speakerLabel)
		}
		p := Proposal{
			ID:                talk.ID,
			Title:             strings.Trim(talk.Title, " "),
			Category:          categories[talk.Categories],
			Format:            formats[talk.Formats],
			Abstract:          talk.Abstract,
			AudienceLevel:     audienceLevels[talk.Level],
			Language:          language,
			Speakers:          strings.Join(speakerLabels, " / "),
			PrivateMessage:    talk.Comments,
			Rating:            talk.Rating,
			Loves:             talk.Loves,
			Hates:             talk.Hates,
			OrganizerMessages: parseOrganizerMessages(talk.OrganizersThread),
		}
		proposals = append(proposals, p)
	}

	return Event{Name: export.Name, Proposals: proposals, Formats: getValues(formats), Categories: getValues(categories)}, nil
}
