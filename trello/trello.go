package trello

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dghubble/oauth1"

	"github.com/bdxio/cfp-to-trello/common"
)

const (
	ListSelection = "Sélection"
	ListRefuses   = "Refusés"
)

type Client interface {
	GetOrganization(name string) (Organization, error)
	CreateBoard(org Organization, name string, permLvl PermissionLevel) (Board, error)
	CreateList(name string, board Board) (List, error)
	CreateLabel(name string, board Board, color Color) (Label, error)
	CreateCard(name, desc string, list List, labels []Label) (Card, error)
	CreateComment(text string, card Card) error
	GetBoards(organization Organization, permLvl PermissionLevel) ([]Board, error)
	GetLists(board Board) ([]List, error)
	GetCards(list List) ([]Card, error)
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Label struct {
	ID string `json:"id"`
}

type Card struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Desc     string
	IDLabels []string
}

type PermissionLevel string

const (
	PermissionLevelOrg     PermissionLevel = "org"
	PermissionLevelPublic  PermissionLevel = "public"
	PermissionLevelPrivate PermissionLevel = "private"
)

type Color string

const (
	ColorYellow Color = "yellow"
	ColorPurple Color = "purple"
	ColorBlue   Color = "blue"
	ColorRed    Color = "red"
	ColorGreen  Color = "green"
	ColorOrange Color = "orange"
	ColorBlack  Color = "black"
	ColorSky    Color = "sky"
	ColorLime   Color = "lime"
	ColorPink   Color = "pink"
)

const tokenValidity = 24 * time.Hour * 30

type Auth struct {
	Token       string    `json:"token"`
	TokenSecret string    `json:"token_secret"`
	ExpiresAt   time.Time `json:"expires_at"`
}

var endpoint = oauth1.Endpoint{
	RequestTokenURL: "https://trello.com/1/OAuthGetRequestToken",
	AuthorizeURL:    "https://trello.com/1/OAuthAuthorizeToken",
	AccessTokenURL:  "https://trello.com/1/OAuthGetAccessToken",
}

type APIClient struct {
	httpClient *http.Client
	labels     map[string]Label
	mu         sync.RWMutex
}

func New(consumerKey, consumerSecret string) (*APIClient, error) {
	config := &oauth1.Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		CallbackURL:    "http://localhost:8000",
		Endpoint:       endpoint,
	}

	token, err := getStoredToken()
	if err != nil {
		return nil, err
	}
	if token != nil {
		return newClient(token.Token, token.TokenSecret, config), nil
	}

	requestToken, requestSecret, err := config.RequestToken()
	if err != nil {
		return nil, err
	}

	authorizationURL, err := config.AuthorizationURL(requestToken)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Open this URL in your browser:\n%s\n", fmt.Sprintf("%s&scope=read,write&name=%s", authorizationURL, url.QueryEscape("BDX I/O - CFP to Trello")))

	lis, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := lis.Close(); err != nil {
			log.Printf("error while closing net listener, ignoring it: %v", err)
		}
	}()
	ch := make(chan string)
	chErr := make(chan error)
	go func() {
		err := http.Serve(lis, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, verifier, pErr := oauth1.ParseAuthorizationCallback(request)
			if pErr != nil {
				chErr <- pErr
			}
			writer.WriteHeader(200)
			ch <- verifier
		}))
		if !errors.Is(err, net.ErrClosed) {
			log.Printf("error while serving HTTP, ignoring it: %v", err)
		}
	}()

	var verifier string
	select {
	case verifier = <-ch:
	case err := <-chErr:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, errors.New("did not receive oauth token")
	}

	accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return nil, err
	}

	auth := Auth{Token: accessToken, TokenSecret: accessSecret, ExpiresAt: time.Now().Add(tokenValidity)}
	data, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}
	trelloAuthPath, trelloAuthDir, err := getStoredAuthPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(trelloAuthDir, 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(trelloAuthPath, data, 0600); err != nil {
		return nil, err
	}
	return newClient(accessToken, accessSecret, config), nil
}

func newClient(accessToken, accessSecret string, config *oauth1.Config) *APIClient {
	httpClient := oauth1.NewClient(context.TODO(), config, &oauth1.Token{
		Token:       accessToken,
		TokenSecret: accessSecret,
	})
	client := &APIClient{httpClient: httpClient, labels: make(map[string]Label)}
	return client
}

func getStoredAuthPath() (path, dir string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir = fmt.Sprintf("%s/.config/cfp-to-trello", homeDir)
	path = fmt.Sprintf("%s/trello.json", dir)
	return
}

func getStoredToken() (*oauth1.Token, error) {
	authPath, _, err := getStoredAuthPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(authPath); err == nil {
		content, err := os.ReadFile(authPath)
		if err != nil {
			return nil, err
		}
		var auth Auth
		if err := json.Unmarshal(content, &auth); err != nil {
			return nil, err
		}
		if auth.ExpiresAt.Before(time.Now()) {
			return nil, nil
		}
		return oauth1.NewToken(auth.Token, auth.TokenSecret), nil
	}
	return nil, nil
}

func (c *APIClient) GetOrganization(name string) (Organization, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("https://api.trello.com/1/organizations/%s", name))
	if err != nil {
		return Organization{}, err
	}
	defer resp.Body.Close()
	var org Organization
	if err := common.UnmarshalBody(resp.Body, &org); err != nil {
		return Organization{}, err
	}
	return org, nil
}

func (c *APIClient) CreateBoard(org Organization, name string, permLvl PermissionLevel) (Board, error) {
	postURL, err := url.Parse("https://api.trello.com/1/boards")
	if err != nil {
		return Board{}, err
	}
	values := postURL.Query()
	values.Add("name", name)
	values.Add("defaultLabels", "false")
	values.Add("defaultLists", "false")
	values.Add("idOrganization", org.ID)
	values.Add("prefs_permissionLevel", string(permLvl))
	postURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Post(postURL.String(), "application/json", nil)
	if err != nil {
		return Board{}, err
	}
	defer resp.Body.Close()
	var board Board
	if err := common.UnmarshalBody(resp.Body, &board); err != nil {
		return Board{}, err
	}
	return board, nil
}

func (c *APIClient) CreateList(name string, board Board) (List, error) {
	postURL, err := url.Parse("https://api.trello.com/1/lists")
	if err != nil {
		return List{}, err
	}
	values := postURL.Query()
	values.Add("name", name)
	values.Add("idBoard", board.ID)
	values.Add("pos", "bottom")
	postURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Post(postURL.String(), "application/json", nil)
	if err != nil {
		return List{}, err
	}
	defer resp.Body.Close()
	var list List
	if err := common.UnmarshalBody(resp.Body, &list); err != nil {
		return List{}, err
	}
	return list, nil
}

func (c *APIClient) CreateLabel(name string, board Board, color Color) (Label, error) {
	// labels are specific to a board
	labelKey := board.ID + "|" + name
	c.mu.RLock()
	label, ok := c.labels[labelKey]
	c.mu.RUnlock()
	if ok {
		return label, nil
	}
	postURL, err := url.Parse("https://api.trello.com/1/labels")
	if err != nil {
		return Label{}, err
	}
	values := postURL.Query()
	values.Add("name", name)
	values.Add("color", string(color))
	values.Add("idBoard", board.ID)
	postURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Post(postURL.String(), "application/json", nil)
	if err != nil {
		return Label{}, err
	}
	defer resp.Body.Close()
	if err := common.UnmarshalBody(resp.Body, &label); err != nil {
		return Label{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.labels[labelKey] = label
	return label, nil
}

func (c *APIClient) CreateCard(name, desc string, list List, labels []Label) (Card, error) {
	postURL, err := url.Parse("https://api.trello.com/1/cards")
	if err != nil {
		return Card{}, err
	}
	values := postURL.Query()
	values.Add("name", name)
	values.Add("desc", desc)
	values.Add("idList", list.ID)
	for _, label := range labels {
		values.Add("idLabels", label.ID)
	}
	postURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Post(postURL.String(), "application/json", nil)
	if err != nil {
		return Card{}, err
	}
	defer resp.Body.Close()
	var card Card
	if err := common.UnmarshalBody(resp.Body, &card); err != nil {
		return Card{}, err
	}
	return card, nil
}

func (c *APIClient) CreateComment(text string, card Card) error {
	postURL, err := url.Parse(fmt.Sprintf("https://api.trello.com/1/cards/%s/actions/comments", card.ID))
	if err != nil {
		return err
	}
	values := postURL.Query()
	values.Add("text", text)
	postURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Post(postURL.String(), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *APIClient) GetBoards(organization Organization, permLvl PermissionLevel) ([]Board, error) {
	getURL, err := url.Parse(fmt.Sprintf("https://api.trello.com/1/organizations/%s/boards", organization.ID))
	if err != nil {
		return nil, err
	}
	values := getURL.Query()
	values.Add("filter", string(permLvl))
	values.Add("fields", "id")
	values.Add("fields", "name")
	getURL.RawQuery = values.Encode()
	resp, err := c.httpClient.Get(getURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var boards []Board
	if err := common.UnmarshalBody(resp.Body, &boards); err != nil {
		return nil, err
	}
	return boards, nil
}

func (c *APIClient) GetLists(board Board) ([]List, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("https://api.trello.com/1/boards/%s/lists", board.ID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var lists []List
	if err := common.UnmarshalBody(resp.Body, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

func (c *APIClient) GetCards(list List) ([]Card, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("https://api.trello.com/1/lists/%s/cards", list.ID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var cards []Card
	if err := common.UnmarshalBody(resp.Body, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

type FakeClient struct {
	Boards   map[string][]List
	Lists    map[string][]Card
	Labels   map[string]Color
	Cards    map[string]Card
	Comments map[string][]string
}

func NewFakeClient() FakeClient {
	return FakeClient{
		Boards:   make(map[string][]List),
		Lists:    make(map[string][]Card),
		Labels:   make(map[string]Color),
		Cards:    make(map[string]Card),
		Comments: make(map[string][]string),
	}
}

func (c FakeClient) GetOrganization(name string) (Organization, error) {
	return Organization{ID: name, Name: name}, nil
}

func (c FakeClient) CreateBoard(_ Organization, name string, _ PermissionLevel) (Board, error) {
	board := Board{ID: name, URL: fmt.Sprintf("http://trello.localhost/%s", name)}
	c.Boards[board.ID] = make([]List, 0)
	return board, nil
}

func (c FakeClient) CreateList(name string, board Board) (List, error) {
	if _, ok := c.Boards[board.ID]; !ok {
		return List{}, fmt.Errorf("board %s doesn't exist", board.ID)
	}
	list := List{ID: fmt.Sprintf("%s-%s", board.ID, name), Name: name}
	c.Lists[list.ID] = make([]Card, 0)
	c.Boards[board.ID] = append(c.Boards[board.ID], list)
	return list, nil
}

func (c FakeClient) CreateLabel(name string, board Board, color Color) (Label, error) {
	if _, ok := c.Boards[board.ID]; !ok {
		return Label{}, fmt.Errorf("board %s doesn't exist", board.ID)
	}
	label := Label{ID: name}
	c.Labels[label.ID] = color
	return label, nil
}

func (c FakeClient) CreateCard(name, desc string, list List, labels []Label) (Card, error) {
	if _, ok := c.Lists[list.ID]; !ok {
		return Card{}, fmt.Errorf("list %s doesn't exist", list.ID)
	}

	// We assume card name (proposal title) to be unique.
	card := Card{ID: name, Name: name, Desc: desc, IDLabels: make([]string, 0, len(labels))}
	for _, label := range labels {
		card.IDLabels = append(card.IDLabels, label.ID)
	}
	c.Lists[list.ID] = append(c.Lists[list.ID], card)
	c.Cards[card.ID] = card
	return card, nil
}

func (c FakeClient) CreateComment(text string, card Card) error {
	if _, ok := c.Cards[card.ID]; !ok {
		return fmt.Errorf("card %s doesn't exist", card.ID)
	}
	if _, ok := c.Comments[card.ID]; !ok {
		c.Comments[card.ID] = make([]string, 0)
	}
	c.Comments[card.ID] = append(c.Comments[card.ID], text)
	return nil
}

func (c FakeClient) GetBoards(_ Organization, _ PermissionLevel) ([]Board, error) {
	boards := make([]Board, 0, len(c.Boards))
	for name := range c.Boards {
		boards = append(boards, Board{ID: name, Name: name})
	}
	return boards, nil
}

func (c FakeClient) GetLists(board Board) ([]List, error) {
	if lists, ok := c.Boards[board.ID]; ok {
		return lists, nil
	}
	return nil, fmt.Errorf("board %s not found", board.ID)
}

func (c FakeClient) GetCards(list List) ([]Card, error) {
	if cards, ok := c.Lists[list.ID]; ok {
		return cards, nil
	}
	return nil, fmt.Errorf("list %s not found", list.ID)
}
