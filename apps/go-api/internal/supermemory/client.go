package supermemory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

const baseURL = "https://api.supermemory.ai"

type Client struct {
	apiKey string
	http   *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != ""
}

type ingestRequest struct {
	Content       string            `json:"content"`
	ContainerTags []string          `json:"containerTags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ingestResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// IngestURL submits a URL (e.g. a presigned R2 URL) for Supermemory to fetch and index.
// containerTag scopes the document for later filtered search.
func (c *Client) IngestURL(ctx context.Context, containerTag, url string, metadata map[string]string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("supermemory: api key not configured")
	}
	body, err := json.Marshal(ingestRequest{
		Content:       url,
		ContainerTags: []string{containerTag},
		Metadata:      metadata,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v3/documents", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("supermemory ingest: %d %s", resp.StatusCode, string(respBody))
	}
	var out ingestResponse
	_ = json.Unmarshal(respBody, &out)
	return out.ID, nil
}

// IngestFile uploads raw bytes (e.g. a PDF) via multipart to Supermemory.
func (c *Client) IngestFile(ctx context.Context, containerTag, filename, contentType string, data []byte, metadata map[string]string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("supermemory: api key not configured")
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := mw.CreatePart(h)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	_ = mw.WriteField("containerTags", containerTag)
	if len(metadata) > 0 {
		if md, err := json.Marshal(metadata); err == nil {
			_ = mw.WriteField("metadata", string(md))
		}
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v3/documents/file", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("supermemory ingest file: %d %s", resp.StatusCode, string(respBody))
	}
	var out ingestResponse
	_ = json.Unmarshal(respBody, &out)
	return out.ID, nil
}

type searchRequest struct {
	Q             string   `json:"q"`
	ContainerTags []string `json:"containerTags,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

type searchResultChunk struct {
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type searchResultDoc struct {
	DocumentID string              `json:"documentId"`
	Score      float64             `json:"score"`
	Chunks     []searchResultChunk `json:"chunks"`
}

type searchResponse struct {
	Results []searchResultDoc `json:"results"`
}

// Search returns concatenated top chunks for the given query within the given container tag.
func (c *Client) Search(ctx context.Context, containerTag, query string, limit int) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("supermemory: api key not configured")
	}
	if limit <= 0 {
		limit = 5
	}
	body, err := json.Marshal(searchRequest{
		Q:             query,
		ContainerTags: []string{containerTag},
		Limit:         limit,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v3/search", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("supermemory search: %d %s", resp.StatusCode, string(respBody))
	}
	var out searchResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	for _, doc := range out.Results {
		for _, ch := range doc.Chunks {
			if buf.Len() > 0 {
				buf.WriteString("\n---\n")
			}
			buf.WriteString(ch.Content)
			if buf.Len() > 4000 {
				return buf.String(), nil
			}
		}
	}
	return buf.String(), nil
}
