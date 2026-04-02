package questionsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type QuestionSet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Length      int    `json:"length"`
}

type Question struct {
	ID            string   `json:"id"`
	Difficulty    int      `json:"difficulty"`
	Categories    []string `json:"categories"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correctAnswer"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: http.DefaultClient,
	}
}

func NewWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) ListSets(ctx context.Context) (sets []QuestionSet, err error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/sets", c.baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build list sets request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load question sets: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close list sets response: %w", closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load question sets: unexpected status %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&sets); err != nil {
		return nil, fmt.Errorf("decode question sets: %w", err)
	}

	return sets, nil
}

func (c *Client) LoadQuestionsBySetID(ctx context.Context, setID string) (questions []Question, err error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/sets/%s", c.baseURL, setID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build questions request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load questions for set %q: %w", setID, err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close questions response for set %q: %w", setID, closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load questions for set %q: unexpected status %d", setID, response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&questions); err != nil {
		return nil, fmt.Errorf("decode questions for set %q: %w", setID, err)
	}

	return questions, nil
}
