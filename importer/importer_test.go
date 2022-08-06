package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bdxio/cfp-to-trello/geo"
	"github.com/bdxio/cfp-to-trello/trello"
)

func TestImportCFP(t *testing.T) {
	client := trello.NewFakeClient()

	err := ImportCFP("test", "123", "../cfp/testdata/export.json", geo.FakeLocate, client)
	require.NoError(t, err)

	// Check boards creation
	assert.Len(t, client.Boards, 1)
	assert.Contains(t, client.Boards, "Délibération Awesome Conference 2042 - Format 1")

	// Check lists creation
	assert.Len(t, client.Lists, 10)
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Sélection")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Désistements")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Backups Acceptés")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Backups")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Category 1 - T1")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Category 1 - T2")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Category 2 - T1")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Category 2 - T2")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-T3")
	assert.Contains(t, client.Lists, "Délibération Awesome Conference 2042 - Format 1-Refusés")

	// Check labels creation
	assert.Len(t, client.Labels, 26)
	// Category labels
	assert.Contains(t, client.Labels, "Category 1")
	assert.Contains(t, client.Labels, "Category 2")
	assert.Equal(t, trello.ColorGreen, client.Labels["Category 1"])
	// Rating labels
	assert.Contains(t, client.Labels, "🏅 2.7")
	assert.Contains(t, client.Labels, "🏅 3.4")
	assert.Contains(t, client.Labels, "🏅 4.1")
	assert.Contains(t, client.Labels, "🏅 3.2")
	assert.Contains(t, client.Labels, "🏅 1.5")
	assert.Equal(t, trello.ColorOrange, client.Labels["🏅 2.7"])
	// Loves/Hates labels
	assert.Contains(t, client.Labels, "0 ❤️ / 0 ☠️")
	assert.Contains(t, client.Labels, "1 ❤️ / 0 ☠️")
	assert.Contains(t, client.Labels, "2 ❤️ / 1 ☠️")
	assert.Contains(t, client.Labels, "2 ❤️ / 0 ☠️")
	assert.Contains(t, client.Labels, "0 ❤️ / 3 ☠️")
	assert.Equal(t, trello.ColorRed, client.Labels["0 ❤️ / 0 ☠️"])
	// Speakers labels
	assert.Contains(t, client.Labels, "Dev from UK - Loveston, UK (Big Bear Stores)")
	assert.Contains(t, client.Labels, "Benjamin Salois - 🗺️ (Wealthy Ideas)")
	assert.Contains(t, client.Labels, "Leala Simard - Carpentras, France (Gold Medal) / Kari Angélil - Muret, France / Anne Course - Lormont, France 🍷")
	assert.Contains(t, client.Labels, "Leala Simard - Carpentras, France (Gold Medal) / Benjamin Salois - 🗺️ (Wealthy Ideas)")
	assert.Contains(t, client.Labels, "Kari Angélil - Muret, France")
	assert.Contains(t, client.Labels, "Leala Simard - Carpentras, France (Gold Medal) / Kari Angélil - Muret, France")
	assert.Contains(t, client.Labels, "Leala Simard - Carpentras, France (Gold Medal)")
	assert.Contains(t, client.Labels, "Leala Simard - Carpentras, France (Gold Medal) / Benjamin Salois - 🗺️ (Wealthy Ideas) / Dev from UK - Loveston, UK (Big Bear Stores)")
	assert.Equal(t, trello.ColorPurple, client.Labels["Dev from UK - Loveston, UK (Big Bear Stores)"])
	// Audience level labels
	assert.Contains(t, client.Labels, "Débutant")
	assert.Contains(t, client.Labels, "Intermédiaire")
	assert.Contains(t, client.Labels, "Avancé")
	assert.Equal(t, trello.ColorSky, client.Labels["Débutant"])
	// Audience level labels
	assert.Contains(t, client.Labels, "🇫🇷")
	assert.Contains(t, client.Labels, "🇬🇧")
	assert.Contains(t, client.Labels, "🇫🇷/🇬🇧")
	assert.Equal(t, trello.ColorPink, client.Labels["🇫🇷"])

	// Check cards creation
	assert.Len(t, client.Cards, 8)
	assert.Contains(t, client.Cards, "A beginner talk in category 1")
	assert.Contains(t, client.Cards, "Another beginner talk in category 1")
	assert.Contains(t, client.Cards, "An intermediate talk in category 1")
	assert.Contains(t, client.Cards, "An advanced talk in category 1")
	assert.Contains(t, client.Cards, "A talk in category 1")
	assert.Contains(t, client.Cards, "A talk in category 2")
	assert.Contains(t, client.Cards, "Another talk in category 2")
	assert.Contains(t, client.Cards, "Still another talk in category 2")
	assert.Equal(
		t,
		trello.Card{
			ID:       "A beginner talk in category 1",
			Name:     "A beginner talk in category 1",
			Desc:     "📜 [Proposal](https://conference-hall.io/organizer/event/123/proposals/6grkSZ4ArcYr8BZfcw0o)\n\n---\n\nAn interesting abstract\n\n---\n\n",
			IDLabels: []string{"Category 1", "🏅 2.7", "0 ❤️ / 0 ☠️", "Leala Simard - Carpentras, France (Gold Medal)", "Débutant", "🇫🇷"},
		},
		client.Cards["A beginner talk in category 1"],
	)

	// Check cards in lists
	assert.Len(t, client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T1"], 2)
	assert.Equal(t, "An advanced talk in category 1", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T1"][0].Name)
	assert.Equal(t, "An intermediate talk in category 1", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T1"][1].Name)
	assert.Len(t, client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T2"], 2)
	assert.Equal(t, "Another beginner talk in category 1", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T2"][0].Name)
	assert.Equal(t, "A talk in category 1", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 1 - T2"][1].Name)
	assert.Len(t, client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 2 - T1"], 1)
	assert.Equal(t, "Another talk in category 2", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 2 - T1"][0].Name)
	assert.Len(t, client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 2 - T2"], 1)
	assert.Equal(t, "A talk in category 2", client.Lists["Délibération Awesome Conference 2042 - Format 1-Category 2 - T2"][0].Name)
	assert.Len(t, client.Lists["Délibération Awesome Conference 2042 - Format 1-T3"], 2)
	assert.Equal(t, "A beginner talk in category 1", client.Lists["Délibération Awesome Conference 2042 - Format 1-T3"][0].Name)
	assert.Equal(t, "Still another talk in category 2", client.Lists["Délibération Awesome Conference 2042 - Format 1-T3"][1].Name)

	// Check comments creation
	assert.Len(t, client.Comments, 1)
	assert.Contains(t, client.Comments, "Another beginner talk in category 1")
	assert.Exactly(
		t,
		[]string{"Second message from another organizer\n--\n**Orga Two** _le 04/08 à 11h46_", "First message from an organizer\n--\n**Orga One** _le 04/08 à 11h44_"},
		client.Comments["Another beginner talk in category 1"],
	)
}
