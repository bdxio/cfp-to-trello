package publisher

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bdxio/cfp-to-trello/cfp"
	"github.com/bdxio/cfp-to-trello/trello"
)

type Publication string

const (
	PublicationAccept Publication = "accept"
	PublicationReject Publication = "reject"
)

func Publish(orgName string, cfpClient cfp.ConferenceHallClient, trelloClient trello.Client, pub Publication) error {
	export, err := cfpClient.GetExport()
	if err != nil {
		return err
	}

	organization, err := trelloClient.GetOrganization(orgName)
	if err != nil {
		return err
	}

	boards, err := trelloClient.GetBoards(organization, trello.PermissionLevelOrg)
	if err != nil {
		return err
	}

	boards = filterTrelloBoards(boards, export.Name, export.Formats)
	if len(boards) == 0 {
		return errors.New("no board for CFP found in Trello")
	}
	for _, board := range boards {
		lists, err := trelloClient.GetLists(board)
		if err != nil {
			return err
		}
		name := trello.ListSelection
		if pub == PublicationReject {
			name = trello.ListRefuses
		}
		list, ok := getList(lists, name)
		if !ok {
			return fmt.Errorf("list %s not found for board %s", name, board.Name)
		}
		cards, err := trelloClient.GetCards(list)
		if err != nil {
			return err
		}
		talks, err := getTalks(cards, export.Talks)
		if err != nil {
			return err
		}
		for _, talk := range talks {
			if !talk.IsSubmitted() {
				log.Printf("Talk is already %s\n", talk.State)
				continue
			}
			log.Printf("%sing talk %s...", pub, talk.Title)
			switch pub {
			case PublicationAccept:
				resp, err := cfpClient.Accept(talk)
				if err != nil {
					return err
				}
				log.Printf("%s\n", resp)
			case PublicationReject:
				resp, err := cfpClient.Reject(talk)
				if err != nil {
					return err
				}
				log.Printf("%s\n", resp)
			}
		}
	}
	return nil
}

func filterTrelloBoards(boards []trello.Board, eventName string, formats []cfp.Format) []trello.Board {
	boardNames := make(map[string]struct{})
	for _, format := range formats {
		boardName := fmt.Sprintf("Délibération %s - %s", eventName, format.Name)
		boardNames[boardName] = struct{}{}
	}
	cfpBoards := make([]trello.Board, 0)
	for _, board := range boards {
		if _, ok := boardNames[board.Name]; ok {
			cfpBoards = append(cfpBoards, board)
		}
	}
	return cfpBoards
}

func getList(lists []trello.List, name string) (list trello.List, ok bool) {
	for _, l := range lists {
		if l.Name == name {
			return l, true
		}
	}
	return
}

func getTalks(cards []trello.Card, talks []cfp.Talk) (matched []cfp.Talk, err error) {
	talksByTitle := make(map[string]cfp.Talk, len(talks))
	for _, talk := range talks {
		title := strings.Trim(talk.Title, " ")
		talksByTitle[title] = talk
	}

	for _, card := range cards {
		if talk, ok := talksByTitle[card.Name]; ok {
			matched = append(matched, talk)
			continue
		}
		return matched, fmt.Errorf("talk %q not found in CFP talks", card.Name)
	}
	return
}

