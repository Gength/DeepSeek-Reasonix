package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"reasonix/internal/tool"
)

func init() { tool.RegisterBuiltin(webSearch{}) }

// Default DeepSeek Anthropic-compatible endpoint for web_search.
const defaultWebSearchBaseURL = "https://api.deepseek.com/anthropic"

const webSearchSystemPrompt = `You are a helpful assistant with access to real-time web search capabilities. When a user asks a question that might benefit from current information, use the web_search tool to find relevant results. After receiving search results, synthesize them into a comprehensive answer with proper citations.`

const webSearchMaxTokens = 32768

type webSearch struct {
	apiKey  string
	baseURL string
	model   string
}

func (webSearch) Name() string { return "web_search" }

func (webSearch) Description() string {
	return `Search the web using the current model provider's native search capability.

Supports DeepSeek's built-in web_search tool (web_search_20250305). When the
current provider is not DeepSeek, this tool returns a clear error suggesting
alternatives (web_fetch or switching to a DeepSeek model).

使用当前模型供应商的原生搜索能力进行网络搜索。
当前支持 DeepSeek 内置的 web_search 工具。如果不是 DeepSeek 模型，
将返回明确的错误提示，建议使用 web_fetch 或切换到 DeepSeek 模型。`
}

func (webSearch) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"Search query / 搜索关键词"},
  "explanation":{"type":"string","description":"Why this search is needed / 为什么需要这个搜索（可选）"}
},
"required":["query"]
}`)
}

func (webSearch) ReadOnly() bool { return true }

func (webSearch) SnipHint() tool.SnipHint {
	return tool.SnipHint{Head: 120, Tail: 12, HeadChars: 12000, TailChars: 2000}
}

// deepSeekAnthropicRequest is the request body for DeepSeek's Anthropic-compatible API.
type deepSeekAnthropicRequest struct {
	Model      string               `json:"model"`
	MaxTokens  int                  `json:"max_tokens"`
	System     string               `json:"system,omitempty"`
	Messages   []anthropicMessage   `json:"messages"`
	Tools      []anthropicToolDef   `json:"tools,omitempty"`
	ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicToolDef struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
}

// deepSeekAnthropicResponse is the response body from DeepSeek's Anthropic-compatible API.
type deepSeekAnthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Error   *deepSeekError          `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// web_search tool results come as type="tool_result" or type="web_search_result"
	Results []webSearchResult `json:"results,omitempty"`
}

type webSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type deepSeekError struct {
	Message string `json:"message"`
}

func (ws webSearch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query       string `json:"query"`
		Explanation string `json:"explanation,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if ws.apiKey == "" {
		return "", fmt.Errorf("web_search requires an API key for DeepSeek's native search endpoint. " +
			"The current provider does not support native web_search. " +
			"Use web_fetch to manually fetch URLs, or switch to a DeepSeek model.")
	}

	baseURL := ws.baseURL
	if baseURL == "" {
		baseURL = defaultWebSearchBaseURL
	}
	model := ws.model
	if model == "" {
		model = "deepseek-chat"
	}

	reqBody := deepSeekAnthropicRequest{
		Model:     model,
		MaxTokens: webSearchMaxTokens,
		System:    webSearchSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: p.Query},
		},
		Tools: []anthropicToolDef{
			{Type: "web_search_20250305", Name: "web_search"},
		},
		ToolChoice: &anthropicToolChoice{Type: "auto"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ws.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_search request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp deepSeekAnthropicResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("web_search API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("web_search API error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result deepSeekAnthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return formatWebSearchResult(result, p.Query), nil
}

func formatWebSearchResult(resp deepSeekAnthropicResponse, query string) string {
	var b strings.Builder

	// Collect text and sources
	var texts []string
	var sources []webSearchResult

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_result", "web_search_result":
			// DeepSeek returns search results as tool_result with a results array
			for _, r := range block.Results {
				sources = append(sources, r)
			}
		}
	}

	// Write the AI-generated answer
	if len(texts) > 0 {
		b.WriteString(strings.TrimSpace(strings.Join(texts, "\n\n")))
		b.WriteString("\n\n")
	}

	// Write the source list
	if len(sources) > 0 {
		b.WriteString("---\n### Sources\n\n")
		for i, src := range sources {
			title := strings.TrimSpace(src.Title)
			url := strings.TrimSpace(src.URL)
			desc := strings.TrimSpace(src.Description)
			if title == "" && url == "" {
				continue
			}
			line := fmt.Sprintf("%d. ", i+1)
			if title != "" {
				line += title
				if url != "" {
					line += fmt.Sprintf(" (%s)", url)
				}
			} else {
				line += url
			}
			if desc != "" {
				line += "\n   " + desc
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	result := strings.TrimSpace(b.String())
	if result == "" {
		return fmt.Sprintf("web_search returned no results for query %q.", query)
	}
	return result
}
