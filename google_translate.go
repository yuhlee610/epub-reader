package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGoogleTranslateEndpoint = "https://translate.googleapis.com/translate_a/single"
	defaultTranslationTarget       = "vi"
	googleTranslateTimeout         = 20 * time.Second
	maxTranslateTextLength         = 5000
)

// GoogleTranslateRequest is the Wails-facing request for selected text.
type GoogleTranslateRequest struct {
	Text           string `json:"text"`
	SourceLanguage string `json:"sourceLanguage,omitempty"`
	TargetLanguage string `json:"targetLanguage,omitempty"`
}

// GoogleTranslateResponse is the text displayed in the in-app translation popup.
type GoogleTranslateResponse struct {
	OriginalText     string `json:"originalText"`
	TranslatedText   string `json:"translatedText"`
	SourceLanguage   string `json:"sourceLanguage,omitempty"`
	TargetLanguage   string `json:"targetLanguage"`
	PronunciationIPA string `json:"pronunciationIpa,omitempty"`
	TextPreview      string `json:"textPreview"`
}

// TranslateSelectedText returns an in-app Google Translate result without
// opening the external Google Translate page.
func (a *App) TranslateSelectedText(request GoogleTranslateRequest) (GoogleTranslateResponse, error) {
	text := cleanPromptLine(request.Text)
	if text == "" {
		return GoogleTranslateResponse{}, errors.New("selected text is required")
	}
	if len(text) > maxTranslateTextLength {
		text = strings.TrimSpace(text[:maxTranslateTextLength])
	}

	sourceLanguage := strings.TrimSpace(request.SourceLanguage)
	if sourceLanguage == "" {
		sourceLanguage = "auto"
	}
	targetLanguage := strings.TrimSpace(request.TargetLanguage)
	if targetLanguage == "" {
		targetLanguage = defaultTranslationTarget
	}

	client := a.httpClient
	if client == nil {
		client = &http.Client{Timeout: googleTranslateTimeout}
	}
	endpoint := strings.TrimSpace(a.googleTranslateEndpoint)
	if endpoint == "" {
		endpoint = defaultGoogleTranslateEndpoint
	}

	translatedText, detectedLanguage, err := sendGoogleTranslateRequest(
		context.Background(),
		client,
		endpoint,
		sourceLanguage,
		targetLanguage,
		text,
	)
	if err != nil {
		return GoogleTranslateResponse{}, err
	}

	return GoogleTranslateResponse{
		OriginalText:     text,
		TranslatedText:   translatedText,
		SourceLanguage:   detectedLanguage,
		TargetLanguage:   targetLanguage,
		PronunciationIPA: a.lookupPronunciationIPA(context.Background(), text),
		TextPreview:      selectedTextPreview(text),
	}, nil
}

func sendGoogleTranslateRequest(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	sourceLanguage string,
	targetLanguage string,
	text string,
) (string, string, error) {
	translateURL, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("parse google translate endpoint: %w", err)
	}

	query := translateURL.Query()
	query.Set("client", "gtx")
	query.Set("sl", sourceLanguage)
	query.Set("tl", targetLanguage)
	query.Set("dt", "t")
	query.Set("q", text)
	translateURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, translateURL.String(), nil)
	if err != nil {
		return "", "", fmt.Errorf("create google translate request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("send google translate request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return "", "", fmt.Errorf("read google translate response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("google translate request failed: status %d", response.StatusCode)
	}

	translatedText, detectedLanguage, err := parseGoogleTranslateResponse(responseBody)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(translatedText) == "" {
		return "", "", errors.New("google translate response did not include translated text")
	}

	return translatedText, detectedLanguage, nil
}

func parseGoogleTranslateResponse(data []byte) (string, string, error) {
	var payload []any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", fmt.Errorf("decode google translate response: %w", err)
	}
	if len(payload) == 0 {
		return "", "", errors.New("google translate response is empty")
	}

	segments, ok := payload[0].([]any)
	if !ok {
		return "", "", errors.New("google translate response has unexpected shape")
	}

	var translated strings.Builder
	for _, segment := range segments {
		values, ok := segment.([]any)
		if !ok || len(values) == 0 {
			continue
		}
		part, ok := values[0].(string)
		if !ok {
			continue
		}
		translated.WriteString(part)
	}

	detectedLanguage := ""
	if len(payload) > 2 {
		detectedLanguage, _ = payload[2].(string)
	}

	return strings.TrimSpace(translated.String()), strings.TrimSpace(detectedLanguage), nil
}
