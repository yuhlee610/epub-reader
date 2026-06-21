package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const (
	defaultDictionaryEndpoint = "https://api.dictionaryapi.dev/api/v2/entries/en/"
	dictionaryLookupTimeout   = 10 * time.Second
)

type dictionaryEntry struct {
	Phonetic  string               `json:"phonetic"`
	Phonetics []dictionaryPhonetic `json:"phonetics"`
}

type dictionaryPhonetic struct {
	Text string `json:"text"`
}

type pronunciationCandidate struct {
	Word      string
	SuffixIPA string
}

func (a *App) lookupPronunciationIPA(ctx context.Context, text string) string {
	word := pronunciationLookupWord(text)
	if word == "" {
		return ""
	}

	client := a.httpClient
	if client == nil {
		client = &http.Client{Timeout: dictionaryLookupTimeout}
	}
	endpoint := strings.TrimSpace(a.dictionaryEndpoint)
	if endpoint == "" {
		endpoint = defaultDictionaryEndpoint
	}

	for _, candidate := range pronunciationLookupCandidates(word) {
		ipa, err := fetchPronunciationIPA(ctx, client, endpoint, candidate.Word)
		if err != nil || ipa == "" {
			continue
		}
		return appendIPASuffix(ipa, candidate.SuffixIPA)
	}

	return ""
}

func fetchPronunciationIPA(ctx context.Context, client *http.Client, endpoint, word string) (string, error) {
	lookupURL := strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(word)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lookupURL, nil)
	if err != nil {
		return "", fmt.Errorf("create dictionary request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send dictionary request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("dictionary request failed: status %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read dictionary response: %w", err)
	}

	var entries []dictionaryEntry
	if err := json.Unmarshal(responseBody, &entries); err != nil {
		return "", fmt.Errorf("decode dictionary response: %w", err)
	}

	return firstPronunciationIPA(entries), nil
}

func firstPronunciationIPA(entries []dictionaryEntry) string {
	for _, entry := range entries {
		for _, phonetic := range entry.Phonetics {
			if ipa := cleanIPA(phonetic.Text); ipa != "" {
				return ipa
			}
		}
		if ipa := cleanIPA(entry.Phonetic); ipa != "" {
			return ipa
		}
	}

	return ""
}

func cleanIPA(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") {
		return value
	}
	return "/" + strings.Trim(value, "/") + "/"
}

func appendIPASuffix(ipa, suffix string) string {
	ipa = cleanIPA(ipa)
	if ipa == "" || suffix == "" {
		return ipa
	}

	return "/" + strings.Trim(ipa, "/") + suffix + "/"
}

func pronunciationLookupCandidates(word string) []pronunciationCandidate {
	candidates := []pronunciationCandidate{{Word: word}}
	if strings.HasSuffix(word, "ing") && len(word) > 4 {
		stem := strings.TrimSuffix(word, "ing")
		addPronunciationCandidate(&candidates, stem, "\u026a\u014b")
		addPronunciationCandidate(&candidates, undoubleFinalLetter(stem), "\u026a\u014b")
		addPronunciationCandidate(&candidates, stem+"e", "\u026a\u014b")
	}
	if strings.HasSuffix(word, "ed") && len(word) > 3 {
		stem := strings.TrimSuffix(word, "ed")
		addPronunciationCandidate(&candidates, stem, "d")
		addPronunciationCandidate(&candidates, undoubleFinalLetter(stem), "d")
		addPronunciationCandidate(&candidates, stem+"e", "d")
	}
	if strings.HasSuffix(word, "s") && len(word) > 3 {
		addPronunciationCandidate(&candidates, strings.TrimSuffix(word, "s"), "z")
	}

	return candidates
}

func addPronunciationCandidate(candidates *[]pronunciationCandidate, word, suffix string) {
	if word == "" {
		return
	}
	for _, candidate := range *candidates {
		if candidate.Word == word && candidate.SuffixIPA == suffix {
			return
		}
	}
	*candidates = append(*candidates, pronunciationCandidate{Word: word, SuffixIPA: suffix})
}

func undoubleFinalLetter(word string) string {
	runes := []rune(word)
	if len(runes) < 2 || runes[len(runes)-1] != runes[len(runes)-2] {
		return word
	}

	return string(runes[:len(runes)-1])
}

func pronunciationLookupWord(text string) string {
	text = strings.TrimSpace(strings.NewReplacer(
		"\u2019", "'",
		"\u2018", "'",
		"\u201c", "",
		"\u201d", "",
		`"`, "",
	).Replace(text))
	if text == "" || strings.ContainsAny(text, " \t\r\n") {
		return ""
	}
	for _, r := range text {
		if unicode.IsDigit(r) {
			return ""
		}
	}

	text = strings.TrimFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\'' && r != '-'
	})
	if text == "" {
		return ""
	}

	for _, r := range text {
		if unicode.IsLetter(r) || r == '\'' || r == '-' {
			continue
		}
		return ""
	}

	return strings.ToLower(text)
}
