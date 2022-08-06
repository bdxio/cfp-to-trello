package cfp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bdxio/cfp-to-trello/geo"
)

func TestParse(t *testing.T) {
	event, err := Parse("testdata/export.json", geo.FakeLocate)

	require.NoError(t, err)
	assert.Equal(t, "Awesome Conference 2042", event.Name)
	assert.Len(t, event.Formats, 2)
	assert.Len(t, event.Categories, 2)
	assert.Len(t, event.Proposals, 8)

	tests := []struct {
		name              string
		id                string
		title             string
		category          string
		format            string
		abstract          string
		audienceLevel     string
		language          string
		speakers          string
		privateMessage    string
		rating            float64
		loves             int
		hates             int
		organizerMessages []string
	}{
		{
			name:              "Talk",
			id:                "6grkSZ4ArcYr8BZfcw0o",
			title:             "A beginner talk in category 1",
			category:          "Category 1",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Débutant",
			language:          "🇫🇷",
			speakers:          "Leala Simard - Carpentras, France (Gold Medal)",
			privateMessage:    "",
			rating:            2.6666666666666665,
			loves:             0,
			hates:             0,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with organizers threads",
			id:                "tsVw51wQQatiEsWzmWfx",
			title:             "Another beginner talk in category 1",
			category:          "Category 1",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Débutant",
			language:          "🇫🇷",
			speakers:          "Kari Angélil - Muret, France",
			privateMessage:    "Speaker with no company and some organizers threads",
			rating:            3.4,
			loves:             0,
			hates:             0,
			organizerMessages: []string{"Second message from another organizer\n--\n**Orga Two** _le 04/08 à 11h46_", "First message from an organizer\n--\n**Orga One** _le 04/08 à 11h44_"},
		},
		{
			name:              "Talk with speaker without address",
			id:                "bSKbIciG4jCWk37vrTEp",
			title:             "An intermediate talk in category 1",
			category:          "Category 1",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Intermédiaire",
			language:          "🇫🇷",
			speakers:          "Benjamin Salois - 🗺️ (Wealthy Ideas)",
			privateMessage:    "Speaker without address",
			rating:            3.4,
			loves:             1,
			hates:             0,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with speaker from another country",
			id:                "Hj2ZNh7ydvOnpg9TBHeL",
			title:             "An advanced talk in category 1",
			category:          "Category 1",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Avancé",
			language:          "🇬🇧",
			speakers:          "Dev from UK - Loveston, UK (Big Bear Stores)",
			privateMessage:    "Speaker from another country",
			rating:            4.123,
			loves:             0,
			hates:             0,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with two speakers, one without company and multiple languages",
			id:                "kZvDMmIaTnrFxGjJycqx",
			title:             "A talk in category 1",
			category:          "Category 1",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Avancé",
			language:          "🇫🇷/🇬🇧",
			speakers:          "Leala Simard - Carpentras, France (Gold Medal) / Kari Angélil - Muret, France",
			privateMessage:    "Two speakers and multiple languages",
			rating:            3.25,
			loves:             0,
			hates:             0,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with two speakers, one without address",
			id:                "tzdLHxKDtVUXcJLd66TN",
			title:             "A talk in category 2",
			category:          "Category 2",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Avancé",
			language:          "🇫🇷",
			speakers:          "Leala Simard - Carpentras, France (Gold Medal) / Benjamin Salois - 🗺️ (Wealthy Ideas)",
			privateMessage:    "Two speakers, one without address",
			rating:            4.123,
			loves:             2,
			hates:             1,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with three speakers",
			id:                "dghzra8K2TfMYnBDjUEb",
			title:             "Another talk in category 2",
			category:          "Category 2",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Avancé",
			language:          "🇫🇷",
			speakers:          "Leala Simard - Carpentras, France (Gold Medal) / Kari Angélil - Muret, France / Anne Course - Lormont, France 🍷",
			privateMessage:    "Three speakers",
			rating:            4.123,
			loves:             2,
			hates:             0,
			organizerMessages: []string{},
		},
		{
			name:              "Talk with three speakers, one without address and one from another country",
			id:                "xdUotyrnjlJ0XiIUZasR",
			title:             "Still another talk in category 2",
			category:          "Category 2",
			format:            "Format 1",
			abstract:          "An interesting abstract",
			audienceLevel:     "Débutant",
			language:          "🇫🇷",
			speakers:          "Leala Simard - Carpentras, France (Gold Medal) / Benjamin Salois - 🗺️ (Wealthy Ideas) / Dev from UK - Loveston, UK (Big Bear Stores)",
			privateMessage:    "Three speakers, one without address and one from another country",
			rating:            1.5,
			loves:             0,
			hates:             3,
			organizerMessages: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proposal, ok := findProposal(event.Proposals, tc.id)

			require.True(t, ok)
			assert.Equal(t, tc.title, proposal.Title)
			assert.Equal(t, tc.category, proposal.Category)
			assert.Equal(t, tc.format, proposal.Format)
			assert.Equal(t, tc.abstract, proposal.Abstract)
			assert.Equal(t, tc.audienceLevel, proposal.AudienceLevel)
			assert.Equal(t, tc.language, proposal.Language)
			assert.Equal(t, tc.speakers, proposal.Speakers)
			assert.Equal(t, tc.privateMessage, proposal.PrivateMessage)
			assert.Equal(t, tc.rating, proposal.Rating)
			assert.Equal(t, tc.loves, proposal.Loves)
			assert.Equal(t, tc.hates, proposal.Hates)
			assert.Equal(t, tc.organizerMessages, proposal.OrganizerMessages)
		})
	}
}

func findProposal(proposals []Proposal, id string) (proposal Proposal, ok bool) {
	for _, p := range proposals {
		if p.ID == id {
			proposal = p
			return proposal, true
		}
	}
	return proposal, false
}

func TestEvent_GetProposals(t *testing.T) {
	event, err := Parse("testdata/export.json", geo.FakeLocate)
	require.NoError(t, err)

	proposals := event.GetProposals("Format 1")
	assert.Len(t, proposals, 8)

	proposals = event.GetProposals("Unknown")
	assert.Empty(t, proposals)
}

func TestEvent_GetProposalsByCategory(t *testing.T) {
	event, err := Parse("testdata/export.json", geo.FakeLocate)
	require.NoError(t, err)

	proposals := event.GetProposalsByCategory("Format 1")

	assert.Len(t, proposals["Category 1"], 5)
	assert.Len(t, proposals["Category 2"], 3)
	assert.Empty(t, proposals["Unknown"])

	proposals = event.GetProposalsByCategory("Format 2")
	assert.Empty(t, proposals)
}
