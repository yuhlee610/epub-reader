package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"epub-reader/internal/library"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultGeminiEndpoint       = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent"
	defaultGeminiStreamEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:streamGenerateContent?alt=sse"
	geminiStreamEventName       = "gemini:study-stream"
	geminiRequestTimeout        = 45 * time.Second
	maxSelectedTextLength       = 6000
)

// GeminiStudyRequest is the Wails-facing request for one selected-text prompt.
type GeminiStudyRequest struct {
	RequestID    string `json:"requestId,omitempty"`
	BookID       string `json:"bookId"`
	PromptID     string `json:"promptId"`
	SelectedText string `json:"selectedText"`
	ChapterTitle string `json:"chapterTitle,omitempty"`
}

// GeminiStudyResponse is the text displayed in the in-app study popup.
type GeminiStudyResponse struct {
	PromptID    string `json:"promptId"`
	PromptName  string `json:"promptName"`
	Text        string `json:"text"`
	UsedModel   string `json:"usedModel,omitempty"`
	TextPreview string `json:"textPreview"`
}

// GeminiStudyStreamEvent is emitted to the frontend while Gemini streams text.
type GeminiStudyStreamEvent struct {
	RequestID string `json:"requestId"`
	Type      string `json:"type"`
	PromptID  string `json:"promptId,omitempty"`
	Text      string `json:"text,omitempty"`
	Error     string `json:"error,omitempty"`
}

type geminiGenerateRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// SendGeminiStudyPrompt sends selected reader text to Gemini using the
// GOOGLE_API_KEY environment variable and returns the generated study response.
func (a *App) SendGeminiStudyPrompt(request GeminiStudyRequest) (GeminiStudyResponse, error) {
	return a.sendGeminiStudyPrompt(request, false)
}

// StreamGeminiStudyPrompt sends selected reader text to Gemini and emits
// gemini:study-stream events as response chunks arrive.
func (a *App) StreamGeminiStudyPrompt(request GeminiStudyRequest) (GeminiStudyResponse, error) {
	return a.sendGeminiStudyPrompt(request, true)
}

func (a *App) sendGeminiStudyPrompt(request GeminiStudyRequest, stream bool) (GeminiStudyResponse, error) {
	store, err := a.libraryStore()
	if err != nil {
		return GeminiStudyResponse{}, err
	}

	book, err := store.GetBook(request.BookID)
	if err != nil {
		return GeminiStudyResponse{}, err
	}

	prompt, err := promptByID(book.Prompt.Prompts, request.PromptID)
	if err != nil {
		return GeminiStudyResponse{}, err
	}

	apiKey := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if apiKey == "" {
		return GeminiStudyResponse{}, errors.New("GOOGLE_API_KEY is not configured")
	}

	endpoint := a.geminiEndpoint
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultGeminiEndpoint
	}
	client := a.httpClient
	if client == nil {
		client = &http.Client{Timeout: geminiRequestTimeout}
	}

	promptText := BuildGeminiStudyPrompt(book, prompt, request.ChapterTitle, request.SelectedText)
	var responseText string
	if stream {
		streamEndpoint := geminiStreamEndpoint(endpoint)
		responseText, err = sendGeminiGenerateContentStream(context.Background(), client, streamEndpoint, apiKey, promptText, func(delta string) {
			a.emitGeminiStudyStreamEvent(GeminiStudyStreamEvent{
				RequestID: request.RequestID,
				Type:      "chunk",
				PromptID:  prompt.ID,
				Text:      delta,
			})
		})
	} else {
		responseText, err = sendGeminiGenerateContent(context.Background(), client, endpoint, apiKey, promptText)
	}
	if err != nil {
		if stream {
			a.emitGeminiStudyStreamEvent(GeminiStudyStreamEvent{
				RequestID: request.RequestID,
				Type:      "error",
				PromptID:  prompt.ID,
				Error:     err.Error(),
			})
		}
		return GeminiStudyResponse{}, err
	}

	response := GeminiStudyResponse{
		PromptID:    prompt.ID,
		PromptName:  prompt.Name,
		Text:        responseText,
		UsedModel:   "gemini-3.5-flash",
		TextPreview: selectedTextPreview(request.SelectedText),
	}
	if stream {
		a.emitGeminiStudyStreamEvent(GeminiStudyStreamEvent{
			RequestID: request.RequestID,
			Type:      "done",
			PromptID:  prompt.ID,
			Text:      responseText,
		})
	}

	return response, nil
}

// BuildGeminiStudyPrompt creates the exact text sent to Gemini for one prompt.
func BuildGeminiStudyPrompt(book library.BookMetadata, prompt library.StudyPrompt, chapterTitle, selectedText string) string {
	bookTitle := cleanPromptLine(book.Title)
	if bookTitle == "" {
		bookTitle = "the current book"
	}
	author := cleanPromptLine(book.Author)
	if author == "" {
		author = "Unknown or not available"
	}
	chapterTitle = cleanPromptLine(chapterTitle)
	if chapterTitle == "" {
		chapterTitle = "Current chapter"
	}

	text := cleanPromptBlock(selectedText)
	if len(text) > maxSelectedTextLength {
		text = strings.TrimSpace(text[:maxSelectedTextLength]) + "\n\n[Selected text truncated for prompt length.]"
	}

	return strings.Join([]string{
		"You are helping me study and translate a passage from a book.",
		"",
		"Book context:",
		"- Title: " + bookTitle,
		"- Author: " + author,
		"- Chapter: " + chapterTitle,
		"",
		"Selected Gemini prompt:",
		cleanPromptBlock(prompt.Instruction),
		"",
		"Selected passage:",
		text,
		"",
		"Response requirements:",
		"- Respect the book context.",
		"- Keep the response useful for language learning.",
		"- Be concise, structured, and clear.",
	}, "\n")
}

func sendGeminiGenerateContent(ctx context.Context, client *http.Client, endpoint, apiKey, promptText string) (string, error) {
	body, err := json.Marshal(geminiGenerateRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: promptText}},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("encode gemini request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-goog-api-key", apiKey)

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send gemini request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read gemini response: %w", err)
	}

	var payload geminiGenerateResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if payload.Error != nil && strings.TrimSpace(payload.Error.Message) != "" {
			return "", fmt.Errorf("gemini request failed: %s", strings.TrimSpace(payload.Error.Message))
		}
		return "", fmt.Errorf("gemini request failed: status %d", response.StatusCode)
	}

	var text strings.Builder
	for _, candidate := range payload.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) == "" {
				continue
			}
			if text.Len() > 0 {
				text.WriteString("\n")
			}
			text.WriteString(strings.TrimSpace(part.Text))
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("gemini response did not include text")
	}

	return text.String(), nil
}

func sendGeminiGenerateContentStream(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	apiKey string,
	promptText string,
	onDelta func(string),
) (string, error) {
	body, err := json.Marshal(geminiGenerateRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: promptText}},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("encode gemini request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create gemini stream request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-goog-api-key", apiKey)

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send gemini stream request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		if err != nil {
			return "", fmt.Errorf("read gemini stream error: %w", err)
		}
		if message := geminiErrorMessage(responseBody); message != "" {
			return "", fmt.Errorf("gemini stream request failed: %s", message)
		}
		return "", fmt.Errorf("gemini stream request failed: status %d", response.StatusCode)
	}

	var text strings.Builder
	if err := readGeminiSSE(response.Body, func(payload geminiGenerateResponse) error {
		if payload.Error != nil && strings.TrimSpace(payload.Error.Message) != "" {
			return errors.New(strings.TrimSpace(payload.Error.Message))
		}

		for _, candidate := range payload.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text == "" {
					continue
				}
				text.WriteString(part.Text)
				if onDelta != nil {
					onDelta(part.Text)
				}
			}
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("decode gemini stream response: %w", err)
	}

	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("gemini response did not include text")
	}

	return text.String(), nil
}

func readGeminiSSE(reader io.Reader, handle func(geminiGenerateResponse) error) error {
	buffer := bufio.NewReader(reader)
	var dataLines []string

	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		if data == "" || data == "[DONE]" {
			return nil
		}

		var payload geminiGenerateResponse
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return err
		}
		return handle(payload)
	}

	for {
		line, err := buffer.ReadString('\n')
		if err != nil && len(line) == 0 {
			if errors.Is(err, io.EOF) {
				return flush()
			}
			return err
		}

		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if flushErr := flush(); flushErr != nil {
				return flushErr
			}
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if errors.Is(err, io.EOF) {
			return flush()
		}
		if err != nil {
			return err
		}
	}
}

func geminiErrorMessage(data []byte) string {
	var payload geminiGenerateResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	if payload.Error == nil {
		return ""
	}
	return strings.TrimSpace(payload.Error.Message)
}

func geminiStreamEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || endpoint == defaultGeminiEndpoint {
		return defaultGeminiStreamEndpoint
	}
	endpoint = strings.Replace(endpoint, ":generateContent", ":streamGenerateContent", 1)
	if strings.Contains(endpoint, "alt=sse") {
		return endpoint
	}
	if strings.Contains(endpoint, "?") {
		return endpoint + "&alt=sse"
	}
	return endpoint + "?alt=sse"
}

func (a *App) emitGeminiStudyStreamEvent(event GeminiStudyStreamEvent) {
	if a.ctx == nil || strings.TrimSpace(event.RequestID) == "" {
		return
	}

	runtime.EventsEmit(a.ctx, geminiStreamEventName, event)
}

func promptByID(prompts []library.StudyPrompt, id string) (library.StudyPrompt, error) {
	id = strings.TrimSpace(id)
	for _, prompt := range prompts {
		if prompt.ID == id {
			return prompt, nil
		}
	}

	return library.StudyPrompt{}, fmt.Errorf("prompt not found: %s", id)
}

func selectedTextPreview(value string) string {
	text := cleanPromptLine(value)
	if len(text) <= 180 {
		return text
	}
	return strings.TrimSpace(text[:180]) + "..."
}

func cleanPromptLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cleanPromptBlock(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
}
