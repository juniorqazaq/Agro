package analyzer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotRecognized = errors.New("disease not recognized")

type Analyzer interface {
	Analyze(ctx context.Context, image []byte, mimeType string, allowedIDs []string) (string, error)
}

type Gemini struct {
	apiKey  string
	model   string
	client  *http.Client
	retries int
}

func NewGemini(apiKey, model string, timeout time.Duration, retries int) *Gemini {
	return &Gemini{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
		retries: retries,
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MIMEType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *Gemini) Analyze(ctx context.Context, image []byte, mimeType string, allowedIDs []string) (string, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return "", errors.New("GEMINI_API_KEY is not configured")
	}

	prompt := fmt.Sprintf(`Analyze the plant photo and select exactly one disease ID from this allowed list:
%s

Return only the disease ID, with no markdown or explanation.
If the image is not a plant, is unclear, or none of the IDs match, return null.
Never invent an ID.`, strings.Join(allowedIDs, "\n"))

	payload, err := json.Marshal(geminiRequest{Contents: []geminiContent{{Parts: []geminiPart{
		{Text: prompt},
		{InlineData: &geminiInlineData{
			MIMEType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(image),
		}},
	}}}})
	if err != nil {
		return "", fmt.Errorf("build Gemini request: %w", err)
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(g.model),
		url.QueryEscape(g.apiKey),
	)

	var lastErr error
	for attempt := 0; attempt < g.retries; attempt++ {
		diseaseID, retry, err := g.request(ctx, endpoint, payload)
		if err == nil {
			return diseaseID, nil
		}
		lastErr = err
		if !retry || attempt == g.retries-1 {
			break
		}

		delay := time.Duration(attempt+1) * 250 * time.Millisecond
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}
	}

	return "", lastErr
}

func (g *Gemini) request(ctx context.Context, endpoint string, payload []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", false, fmt.Errorf("create Gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("call Gemini: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", true, fmt.Errorf("read Gemini response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return "", retry, fmt.Errorf("Gemini returned status %d", resp.StatusCode)
	}

	var result geminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, fmt.Errorf("decode Gemini response: %w", err)
	}
	if result.Error != nil {
		return "", false, fmt.Errorf("Gemini error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", false, ErrNotRecognized
	}

	answer := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	answer = strings.Trim(answer, "`\"' \r\n\t")
	if strings.EqualFold(answer, "null") || answer == "" {
		return "", false, ErrNotRecognized
	}
	return answer, false, nil
}

type Mock struct {
	DiseaseID string
}

func (m Mock) Analyze(context.Context, []byte, string, []string) (string, error) {
	if m.DiseaseID == "" || strings.EqualFold(m.DiseaseID, "null") {
		return "", ErrNotRecognized
	}
	return m.DiseaseID, nil
}
