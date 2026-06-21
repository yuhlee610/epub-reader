package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranslateSelectedTextCallsGoogleTranslateAndParsesResponse(t *testing.T) {
	var gotSource string
	var gotTarget string
	var gotText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.URL.Query().Get("sl")
		gotTarget = r.URL.Query().Get("tl")
		gotText = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[["Xin chào ","Hello ",null,null,10],[ "thế giới","world",null,null,10]],null,"en"]`))
	}))
	defer server.Close()

	app := &App{
		ctx:                     context.Background(),
		httpClient:              server.Client(),
		googleTranslateEndpoint: server.URL,
	}

	response, err := app.TranslateSelectedText(GoogleTranslateRequest{
		Text:           "Hello world",
		TargetLanguage: "vi",
	})
	if err != nil {
		t.Fatalf("TranslateSelectedText() error = %v", err)
	}
	if gotSource != "auto" {
		t.Fatalf("source language = %q, want auto", gotSource)
	}
	if gotTarget != "vi" {
		t.Fatalf("target language = %q, want vi", gotTarget)
	}
	if gotText != "Hello world" {
		t.Fatalf("text query = %q, want selected text", gotText)
	}
	if response.TranslatedText != "Xin chào thế giới" {
		t.Fatalf("TranslatedText = %q, want parsed translation", response.TranslatedText)
	}
	if response.SourceLanguage != "en" {
		t.Fatalf("SourceLanguage = %q, want en", response.SourceLanguage)
	}
}

func TestTranslateSelectedTextRequiresSelectedText(t *testing.T) {
	app := &App{}

	_, err := app.TranslateSelectedText(GoogleTranslateRequest{})
	if err == nil || !strings.Contains(err.Error(), "selected text") {
		t.Fatalf("TranslateSelectedText() error = %v, want selected text error", err)
	}
}

func TestParseGoogleTranslateResponseRejectsUnexpectedShape(t *testing.T) {
	_, _, err := parseGoogleTranslateResponse([]byte(`{"not":"the expected array"}`))
	if err == nil {
		t.Fatal("parseGoogleTranslateResponse() error = nil, want error")
	}
}
