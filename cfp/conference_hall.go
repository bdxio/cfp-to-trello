package cfp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/bdxio/cfp-to-trello/common"
)

const URL = "https://conference-hall.io"

type ConferenceHallClient struct {
	url     string
	eventID string
	apiKey  string
	client  *http.Client
	dryRun  bool
}

type ConferenceHallClientOption func(client *ConferenceHallClient)

func WithURL(url string) ConferenceHallClientOption {
	return func(c *ConferenceHallClient) {
		c.url = url
	}
}

func WithEventID(eventID string) ConferenceHallClientOption {
	return func(c *ConferenceHallClient) {
		c.eventID = eventID
	}
}

func WithAPIKey(apiKey string) ConferenceHallClientOption {
	return func(c *ConferenceHallClient) {
		c.apiKey = apiKey
	}
}

func WithHTTPClient(client *http.Client) ConferenceHallClientOption {
	return func(c *ConferenceHallClient) {
		c.client = client
	}
}

func WithDryRun(dryRun bool) ConferenceHallClientOption {
	return func(c *ConferenceHallClient) {
		c.dryRun = dryRun
	}
}

func NewConferenceHallClient(opts ...ConferenceHallClientOption) ConferenceHallClient {
	client := ConferenceHallClient{client: http.DefaultClient}
	for _, opt := range opts {
		opt(&client)
	}
	return client
}

func (c ConferenceHallClient) GetExport() (Export, error) {
	getURL, err := url.Parse(fmt.Sprintf("%s/api/v1/event/%s", c.url, c.eventID))
	if err != nil {
		return Export{}, err
	}
	values := getURL.Query()
	values.Add("key", c.apiKey)
	getURL.RawQuery = values.Encode()
	resp, err := c.client.Get(getURL.String())
	if err != nil {
		return Export{}, err
	}
	defer resp.Body.Close()
	var export Export
	if err := common.UnmarshalBody(resp.Body, &export); err != nil {
		return Export{}, err
	}
	return export, nil
}

type talkAction string

const (
	talkAccept talkAction = "accept"
	talkReject talkAction = "reject"
)

func (c ConferenceHallClient) publish(talk Talk, action talkAction) (string, error) {
	putURL, err := url.Parse(fmt.Sprintf("%s/api/v1/proposal/%s/%s/%s", c.url, c.eventID, talk.ID, action))
	if err != nil {
		return "", err
	}
	values := putURL.Query()
	values.Add("key", c.apiKey)
	putURL.RawQuery = values.Encode()
	if c.dryRun {
		log.Printf("%sing talk %q: %s", action, talk.Title, putURL.String())
		return "ok", nil
	}
	req, err := http.NewRequest(http.MethodPut, putURL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error while %sing talk %s: %d", action, talk.Title, resp.StatusCode)
	}

	var jsonResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return "", err
	}
	return jsonResp["result"], nil
}

func (c ConferenceHallClient) Accept(talk Talk) (string, error) {
	return c.publish(talk, talkAccept)
}

func (c ConferenceHallClient) Reject(talk Talk) (string, error) {
	return c.publish(talk, talkReject)
}

type ConferenceHallServer struct {
	URL         string
	Client      *http.Client
	AcceptedIDs []string
	RejectedIDs []string
	eventID     string
	apiKey      string
	jsonPath    string
}

func NewConferenceHallServer(eventID, apiKey, jsonPath string) (*ConferenceHallServer, func()) {
	cfpServer := &ConferenceHallServer{
		AcceptedIDs: make([]string, 0),
		RejectedIDs: make([]string, 0),
		eventID:     eventID,
		apiKey:      apiKey,
		jsonPath:    jsonPath,
	}
	s := httptest.NewTLSServer(cfpServer)
	cfpServer.URL = s.URL
	cfpServer.Client = s.Client()
	return cfpServer, func() { s.Close() }
}

func (s *ConferenceHallServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, "/api/v1/event/"):
		s.sendEvent(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.RequestURI, "/api/v1/proposal/"):
		s.publishEvent(w, r)
	default:
		log.Printf("invalid request: %s\n", r.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *ConferenceHallServer) sendEvent(w http.ResponseWriter, r *http.Request) {
	payload := strings.TrimPrefix(r.RequestURI, "/api/v1/event/")
	paths := strings.Split(payload, "?key=")
	if len(paths) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if paths[0] != s.eventID {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid eventID"))
		return
	}
	if paths[1] != s.apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid API key"))
		return
	}
	data, err := os.ReadFile(s.jsonPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *ConferenceHallServer) publishEvent(w http.ResponseWriter, r *http.Request) {
	accept := strings.Contains(r.RequestURI, "accept")
	payload := strings.TrimPrefix(r.RequestURI, "/api/v1/proposal/")
	paths := strings.Split(payload, "/")
	if len(paths) != 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if paths[0] != s.eventID {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid eventID"))
		return
	}
	parameters := strings.Split(paths[2], "?key=")
	if len(parameters) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if parameters[1] != s.apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid API key"))
		return
	}
	if accept {
		s.AcceptedIDs = append(s.AcceptedIDs, paths[1])
	} else {
		s.RejectedIDs = append(s.RejectedIDs, paths[1])
	}

	result := fmt.Sprintf(`{"result": "Proposal with ID %s is now %sed."}`, paths[1], parameters[0])
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(result))
}
