package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Classification is the result from the classifier service.
type Classification struct {
	Classification string   `json:"classification"`
	Direction      string   `json:"direction"`
	Flags          []string `json:"flags"`
	Score          float64  `json:"score"`
}

// ClassifyRequest is sent to the classifier service.
type ClassifyRequest struct {
	Text      string `json:"text"`
	Direction string `json:"direction"`
}

// Classifier talks to the classifier service.
type Classifier struct {
	client   *http.Client
	endpoint string
}

// NewClassifier creates a classifier client for the given endpoint.
// Supports "unix:///path/to/sock" and "http://host:port" formats.
func NewClassifier(endpoint string) *Classifier {
	c := &Classifier{endpoint: endpoint}

	if strings.HasPrefix(endpoint, "unix://") {
		socketPath := strings.TrimPrefix(endpoint, "unix://")
		c.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		}
		c.endpoint = "http://ice"
	} else {
		c.client = http.DefaultClient
	}

	return c
}

// Classify sends text to the classifier and returns the result.
func (c *Classifier) Classify(text, direction string) (*Classification, error) {
	body, err := json.Marshal(ClassifyRequest{Text: text, Direction: direction})
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := c.client.Post(c.endpoint+"/classify", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("classifier request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("classifier returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result Classification
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// Available reports whether a classifier endpoint is configured.
func Available() bool {
	return ClassifierEndpoint != ""
}

// classifierInstance returns a classifier or nil if unconfigured.
func classifierInstance() *Classifier {
	if !Available() {
		return nil
	}
	return NewClassifier(ClassifierEndpoint)
}
