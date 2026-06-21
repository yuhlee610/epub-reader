package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"epub-reader/internal/library"
)

func TestBuildGeminiStudyPromptIncludesBookPromptAndSelection(t *testing.T) {
	prompt := BuildGeminiStudyPrompt(
		library.BookMetadata{
			Title:  "Cradle",
			Author: "Will Wight",
		},
		library.StudyPrompt{
			Name:        "Grammar",
			Instruction: "Explain grammar patterns.",
		},
		"Chapter 7",
		"Yerin moved quickly.",
	)

	for _, want := range []string{
		"- Title: Cradle",
		"- Author: Will Wight",
		"- Chapter: Chapter 7",
		"Explain grammar patterns.",
		"Yerin moved quickly.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestSendGeminiStudyPromptRequiresGoogleAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	app := newGeminiTestApp(t, nil)

	_, err := app.SendGeminiStudyPrompt(GeminiStudyRequest{
		BookID:       "book-1",
		PromptID:     "translate",
		SelectedText: "Selected text",
	})
	if err == nil || !strings.Contains(err.Error(), "GOOGLE_API_KEY") {
		t.Fatalf("SendGeminiStudyPrompt() error = %v, want GOOGLE_API_KEY error", err)
	}
}

func TestSendGeminiStudyPromptPostsToGeminiAndParsesResponse(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key")

	var gotKey string
	var gotPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-goog-api-key")

		var request struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotPrompt = request.Contents[0].Parts[0].Text

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{"text": "Translated response"}]
				}
			}]
		}`))
	}))
	defer server.Close()

	app := newGeminiTestApp(t, func(app *App) {
		app.geminiEndpoint = server.URL
		app.httpClient = server.Client()
	})

	response, err := app.SendGeminiStudyPrompt(GeminiStudyRequest{
		BookID:       "book-1",
		PromptID:     "grammar",
		SelectedText: "Yerin moved quickly.",
		ChapterTitle: "Chapter 3",
	})
	if err != nil {
		t.Fatalf("SendGeminiStudyPrompt() error = %v", err)
	}
	if gotKey != "test-key" {
		t.Fatalf("x-goog-api-key = %q, want test key", gotKey)
	}
	if !strings.Contains(gotPrompt, "Grammar notes") || !strings.Contains(gotPrompt, "Yerin moved quickly.") {
		t.Fatalf("posted prompt missing expected text:\n%s", gotPrompt)
	}
	if response.Text != "Translated response" {
		t.Fatalf("response.Text = %q, want parsed Gemini text", response.Text)
	}
	if response.PromptName != "Grammar" {
		t.Fatalf("PromptName = %q, want Grammar", response.PromptName)
	}
}

func TestSendGeminiGenerateContentStreamParsesSSEChunks(t *testing.T) {
	var gotAlt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAlt = r.URL.Query().Get("alt")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello \"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"world\"}]}}]}\n\n"))
	}))
	defer server.Close()

	var chunks []string
	text, err := sendGeminiGenerateContentStream(
		context.Background(),
		server.Client(),
		geminiStreamEndpoint(server.URL),
		"test-key",
		"Prompt text",
		func(delta string) {
			chunks = append(chunks, delta)
		},
	)
	if err != nil {
		t.Fatalf("sendGeminiGenerateContentStream() error = %v", err)
	}
	if gotAlt != "sse" {
		t.Fatalf("alt query = %q, want sse", gotAlt)
	}
	if text != "Hello world" {
		t.Fatalf("stream text = %q, want Hello world", text)
	}
	if strings.Join(chunks, "") != "Hello world" {
		t.Fatalf("chunks = %#v, want streamed text", chunks)
	}
}

func TestSendGeminiGenerateContentStreamReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"error\":{\"message\":\"quota exceeded\"}}\n\n"))
	}))
	defer server.Close()

	_, err := sendGeminiGenerateContentStream(
		context.Background(),
		server.Client(),
		server.URL,
		"test-key",
		"Prompt text",
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("sendGeminiGenerateContentStream() error = %v, want quota exceeded", err)
	}
}

func newGeminiTestApp(t *testing.T, configure func(*App)) *App {
	t.Helper()

	store, err := library.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	_, err = store.SaveBook(library.BookMetadata{
		ID:               "book-1",
		Title:            "Cradle",
		Author:           "Will Wight",
		OriginalFileName: "cradle.epub",
		Prompt: library.PromptConfig{
			Prompts: []library.StudyPrompt{
				{
					ID:          "translate",
					Name:        "Translate",
					ShortLabel:  "TR",
					Instruction: "Translate naturally.",
				},
				{
					ID:          "grammar",
					Name:        "Grammar",
					ShortLabel:  "GR",
					Instruction: "Grammar notes",
					SortOrder:   1,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}

	app := &App{
		ctx:          context.Background(),
		store:        store,
		openEPUBFile: func(context.Context) (string, error) { return "", nil },
	}
	if configure != nil {
		configure(app)
	}
	return app
}
