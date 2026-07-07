package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchName(t *testing.T) {
	ws := webSearch{}
	if got := ws.Name(); got != "web_search" {
		t.Errorf("Name() = %q, want %q", got, "web_search")
	}
}

func TestWebSearchReadOnly(t *testing.T) {
	ws := webSearch{}
	if !ws.ReadOnly() {
		t.Error("ReadOnly() should be true")
	}
}

func TestWebSearchSchema(t *testing.T) {
	ws := webSearch{}
	schema := ws.Schema()
	if !strings.Contains(string(schema), "query") {
		t.Error("Schema should contain query parameter")
	}
	if !strings.Contains(string(schema), "explanation") {
		t.Error("Schema should contain explanation parameter")
	}

	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}
}

func TestWebSearchEmptyArgs(t *testing.T) {
	ws := webSearch{apiKey: "test-key"}
	_, err := ws.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for empty args")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Errorf("error = %q, want query required", err.Error())
	}
}

func TestWebSearchInvalidJSON(t *testing.T) {
	ws := webSearch{apiKey: "test-key"}
	_, err := ws.Execute(context.Background(), json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWebSearchNoAPIKey(t *testing.T) {
	ws := webSearch{} // empty apiKey
	_, err := ws.Execute(context.Background(), json.RawMessage(`{"query":"hello"}`))
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "API key") {
		t.Errorf("error should mention missing API key, got: %v", err)
	}
}

func TestWebSearchAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer srv.Close()

	ws := webSearch{apiKey: "bad-key", baseURL: srv.URL}
	_, err := ws.Execute(context.Background(), json.RawMessage(`{"query":"hello"}`))
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("error should contain API error message, got: %v", err)
	}
}

func TestWebSearchSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key header = %q, want test-key", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"content": [
				{"type": "text", "text": "The weather in Siyang is sunny today with a high of 28°C."},
				{"type": "tool_result", "results": [
					{"title": "Siyang Weather", "url": "http://weather.com/siyang", "description": "7-day forecast for Siyang"}
				]}
			]
		}`))
	}))
	defer srv.Close()

	ws := webSearch{apiKey: "test-key", baseURL: srv.URL, model: "deepseek-chat"}
	result, err := ws.Execute(context.Background(), json.RawMessage(`{"query":"Siyang weather"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(result, "28°C") {
		t.Errorf("result should contain weather info, got: %s", result)
	}
	if !strings.Contains(result, "Siyang Weather") {
		t.Errorf("result should contain source title, got: %s", result)
	}
	if !strings.Contains(result, "weather.com/siyang") {
		t.Errorf("result should contain source URL, got: %s", result)
	}
}

func TestWebSearchEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": []}`))
	}))
	defer srv.Close()

	ws := webSearch{apiKey: "test-key", baseURL: srv.URL}
	result, err := ws.Execute(context.Background(), json.RawMessage(`{"query":"nothing"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "nothing") {
		t.Errorf("empty response should mention the query, got: %s", result)
	}
}

func TestConfineWebSearch(t *testing.T) {
	ws := ConfineWebSearch("key", "https://api.deepseek.com/anthropic", "deepseek-chat")
	if ws.Name() != "web_search" {
		t.Errorf("ConfineWebSearch returned tool named %q", ws.Name())
	}
	if !ws.ReadOnly() {
		t.Error("ConfineWebSearch should return read-only tool")
	}
}
