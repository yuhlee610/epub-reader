package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPronunciationIPAParsesDictionaryResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Fatalf("path = %q, want /hello", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"word":"hello","phonetics":[{"text":"/h\u0259\u02c8l\u0259\u028a/"}]}]`))
	}))
	defer server.Close()

	ipa, err := fetchPronunciationIPA(context.Background(), server.Client(), server.URL, "hello")
	if err != nil {
		t.Fatalf("fetchPronunciationIPA() error = %v", err)
	}
	if ipa != "/h\u0259\u02c8l\u0259\u028a/" {
		t.Fatalf("ipa = %q, want /h\u0259\u02c8l\u0259\u028a/", ipa)
	}
}

func TestLookupPronunciationIPAFallsBackForIngForms(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/hauling":
			_, _ = w.Write([]byte(`[{"word":"hauling","phonetics":[]}]`))
		case "/haul":
			_, _ = w.Write([]byte(`[{"word":"haul","phonetics":[{"text":"/h\u0254\u02d0l/"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := &App{
		httpClient:         server.Client(),
		dictionaryEndpoint: server.URL,
	}
	ipa := app.lookupPronunciationIPA(context.Background(), "hauling")
	if ipa != "/h\u0254\u02d0l\u026a\u014b/" {
		t.Fatalf("ipa = %q, want /h\u0254\u02d0l\u026a\u014b/", ipa)
	}
	if len(paths) < 2 || paths[0] != "/hauling" || paths[1] != "/haul" {
		t.Fatalf("lookup paths = %#v, want hauling then haul", paths)
	}
}

func TestPronunciationLookupWordOnlyAcceptsSingleWords(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "plain word", text: "Insane", want: "insane"},
		{name: "apostrophe", text: "hadn\u2019t", want: "hadn't"},
		{name: "punctuation", text: `"hello,"`, want: "hello"},
		{name: "phrase", text: "hello world", want: ""},
		{name: "number", text: "chapter3", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pronunciationLookupWord(tt.text); got != tt.want {
				t.Fatalf("pronunciationLookupWord() = %q, want %q", got, tt.want)
			}
		})
	}
}
