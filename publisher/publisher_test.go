package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bdxio/cfp-to-trello/cfp"
	"github.com/bdxio/cfp-to-trello/trello"
)

func TestPublish_Accept(t *testing.T) {
	// setup Conference Hall
	srv, stop := cfp.NewConferenceHallServer("12345", "67890", "../cfp/testdata/export.json")
	t.Cleanup(stop)
	cfpClient := cfp.NewConferenceHallClient(
		cfp.WithURL(srv.URL),
		cfp.WithEventID("12345"),
		cfp.WithAPIKey("67890"),
		cfp.WithHTTPClient(srv.Client),
	)

	// setup Trello
	trelloClient := setupTrello(t)

	err := Publish("test", cfpClient, trelloClient, PublicationAccept)
	require.NoError(t, err)

	assert.Equal(t, []string{"6grkSZ4ArcYr8BZfcw0o", "dghzra8K2TfMYnBDjUEb"}, srv.AcceptedIDs)
	assert.Empty(t, srv.RejectedIDs)
}

func TestPublish_Reject(t *testing.T) {
	// setup Conference Hall
	srv, stop := cfp.NewConferenceHallServer("12345", "67890", "../cfp/testdata/export.json")
	t.Cleanup(stop)
	cfpClient := cfp.NewConferenceHallClient(
		cfp.WithURL(srv.URL),
		cfp.WithEventID("12345"),
		cfp.WithAPIKey("67890"),
		cfp.WithHTTPClient(srv.Client),
	)

	// setup Trello
	trelloClient := setupTrello(t)

	err := Publish("test", cfpClient, trelloClient, PublicationReject)
	require.NoError(t, err)

	assert.Equal(t, []string{"Hj2ZNh7ydvOnpg9TBHeL", "xdUotyrnjlJ0XiIUZasR"}, srv.RejectedIDs)
	assert.Empty(t, srv.AcceptedIDs)
}

func setupTrello(t *testing.T) trello.Client {
	org := trello.Organization{}
	trelloClient := trello.NewFakeClient()
	// setup board 1
	board1, err := trelloClient.CreateBoard(org, "Délibération Awesome Conference 2042 - Format 1", trello.PermissionLevelOrg)
	require.NoError(t, err)
	listAcceptes1, err := trelloClient.CreateList(trello.ListSelection, board1)
	require.NoError(t, err)
	listBackups1, err := trelloClient.CreateList("Backups", board1)
	require.NoError(t, err)
	listRefuses1, err := trelloClient.CreateList(trello.ListRefuses, board1)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("A beginner talk in category 1", "", listAcceptes1, nil)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("Another talk in category 2", "", listAcceptes1, nil)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("Another beginner talk in category 1", "Already accepted", listAcceptes1, nil)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("A talk in category 2", "", listBackups1, nil)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("An advanced talk in category 1", "", listRefuses1, nil)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("Still another talk in category 2", "", listRefuses1, nil)
	require.NoError(t, err)

	// setup board 2 (no talk in "Acceptés" or "Refusés" lists)
	board2, err := trelloClient.CreateBoard(org, "Délibération Awesome Conference 2042 - Format 2", trello.PermissionLevelOrg)
	require.NoError(t, err)
	_, err = trelloClient.CreateList(trello.ListSelection, board2)
	require.NoError(t, err)
	listBackups2, err := trelloClient.CreateList("Backups", board2)
	require.NoError(t, err)
	_, err = trelloClient.CreateList(trello.ListRefuses, board2)
	require.NoError(t, err)
	_, err = trelloClient.CreateCard("A talk in category 1", "", listBackups2, nil)

	// another unrelated board
	_, err = trelloClient.CreateBoard(org, "Another unrelated board", trello.PermissionLevelOrg)
	require.NoError(t, err)

	return trelloClient
}

func TestPublish_NoTrelloBoard(t *testing.T) {
	// setup Conference Hall
	srv, stop := cfp.NewConferenceHallServer("12345", "67890", "../cfp/testdata/export.json")
	t.Cleanup(stop)
	cfpClient := cfp.NewConferenceHallClient(
		cfp.WithURL(srv.URL),
		cfp.WithEventID("12345"),
		cfp.WithAPIKey("67890"),
		cfp.WithHTTPClient(srv.Client),
	)

	// setup Trello
	trelloClient := trello.NewFakeClient()

	err := Publish("test", cfpClient, trelloClient, PublicationAccept)

	assert.Error(t, err, "no board for CFP found in Trello")
}
