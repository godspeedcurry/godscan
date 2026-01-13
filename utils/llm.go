package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultLLMProvider = "google"
	DefaultLLMModel    = "gemini-2.5-flash"
	DefaultLLMPrompt   = "你是安全分析助手。请以Markdown表格形式输出，按 root_url 逐条输出风险评分(0-10)、关键证据（敏感命中/高熵/重要API/CDN暴露等）、以及明显的误报要排除。要求：\n- 不要泛泛而谈，只引用输入里给出的内容；\n- 先列高风险，再列中低风险；\n- 误报要明确标记；\n- 简短中文要点。"
	DefaultLLMMaxChars = 128000
)

type LLMConfig struct {
	Provider      string
	Model         string
	Prompt        string
	APIKey        string
	BaseURL       string
	MaxInputChars int
	MaxOutputTok  int
	DryRun        bool
}

type LLMSummary struct {
	Enabled      bool   `json:"enabled"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	Output       string `json:"output"`
	Error        string `json:"error,omitempty"`
	InputPreview string `json:"input_preview,omitempty"`
	InputTotal   int    `json:"input_total"`
	InputLimit   int    `json:"input_limit,omitempty"`
	OutputLimit  int    `json:"output_limit,omitempty"`
	GeneratedAt  string `json:"generated_at,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
}

// SummarizeReport builds a compact crawler context and sends it to the configured LLM.
// Failures are captured in the returned summary; they do not block report generation.
func SummarizeReport(ctx context.Context, data HTMLReportData, cfg *LLMConfig) *LLMSummary {
	res := &LLMSummary{
		Enabled:     cfg != nil,
		Provider:    DefaultLLMProvider,
		Model:       DefaultLLMModel,
		Prompt:      DefaultLLMPrompt,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	if cfg == nil {
		return res
	}
	if cfg.Provider != "" {
		res.Provider = strings.ToLower(cfg.Provider)
	}
	if cfg.Model != "" {
		res.Model = cfg.Model
	}
	if cfg.Prompt != "" {
		res.Prompt = cfg.Prompt
	}
	res.DryRun = cfg.DryRun
	maxChars := cfg.MaxInputChars
	if maxChars <= 0 {
		maxChars = DefaultLLMMaxChars
	}
	res.InputLimit = maxChars
	res.OutputLimit = maxOutputTokens(cfg)
	input := buildLLMInput(data, maxChars)
	res.InputTotal = len(input)
	res.InputPreview = previewText(input, 180000)
	if strings.TrimSpace(input) == "" {
		res.Error = "no crawler text available for LLM summary"
		return res
	}
	if cfg.DryRun {
		res.Output = fmt.Sprintf("LLM dry-run: call skipped (payload %d chars).", len(input))
		Info("LLM dry-run preview (provider=%s model=%s chars=%d):\n%s", res.Provider, res.Model, len(input), res.InputPreview)
		return res
	}
	Info("LLM request start provider=%s model=%s chars=%d", res.Provider, res.Model, len(input))
	if ctx == nil {
		ctx = context.Background()
	}

	var out string
	var err error

	// Dispatch based on config
	if cfg.BaseURL != "" || strings.ToLower(cfg.Provider) == "openai" {
		out, err = callOpenAI(ctx, cfg, input)
	} else if strings.ToLower(cfg.Provider) == "google" {
		out, err = callGoogleLLM(ctx, cfg, input)
	} else {
		// Default or unknown -> try generic/OpenAI if we have a BaseURL, else Google?
		// Actually if provider is not "google" (default) but BaseURL is empty, it's an error unless we default to OpenAI for non-google providers.
		// For now, assume unconfigured means Google defaults.
		out, err = callGoogleLLM(ctx, cfg, input)
	}

	if err != nil {
		Warning("LLM request failed: %v", err)
		res.Error = err.Error()
		return res
	}
	res.Output = strings.TrimSpace(out)
	Info("LLM request done provider=%s model=%s output_chars=%d", res.Provider, res.Model, len(res.Output))
	return res
}

func buildLLMInput(data HTMLReportData, maxChars int) string {
	if maxChars <= 0 {
		maxChars = DefaultLLMMaxChars
	}
	var sb strings.Builder
	writeLine := func(line string) bool {
		if line == "" || sb.Len() >= maxChars {
			return sb.Len() < maxChars
		}
		if sb.Len()+len(line)+1 > maxChars {
			line = line[:maxChars-sb.Len()-1]
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
		return sb.Len() < maxChars
	}

	sensByRoot := map[string][]SensitiveHit{}
	for _, s := range data.Sensitive {
		r := rootOf(s.SourceURL)
		if r == "" {
			continue
		}
		sensByRoot[r] = append(sensByRoot[r], s)
	}
	apiByRoot := map[string][]APIPathRow{}
	impByRoot := map[string][]APIPathRow{}
	for _, a := range data.APIs {
		r := a.RootURL
		if r == "" {
			r = rootOf(a.SourceURL)
		}
		if r == "" {
			continue
		}
		apiByRoot[r] = append(apiByRoot[r], a)
		if isImportantAPI(a.Path, data.ImportantAPIs) {
			impByRoot[r] = append(impByRoot[r], a)
		}
	}

	mapsByRoot := map[string]int{}
	for _, m := range data.SourceMaps {
		mapsByRoot[m.RootURL]++
	}

	bodiesByRoot := map[string][]PageSnapshotLite{}
	for _, p := range data.PageBodies {
		r := p.RootURL
		if r == "" {
			r = rootOf(p.URL)
		}
		if r != "" {
			bodiesByRoot[r] = append(bodiesByRoot[r], p)
		}
	}

	writeLine("# Per-root findings (focused on sensitive/high-entropy/CDN/important APIs)")
	type rootScore struct {
		rec   SpiderRecord
		score int
	}
	var scored []rootScore
	for _, s := range data.Summary {
		sig := len(sensByRoot[s.Url])*3 + len(impByRoot[s.Url])*2
		if s.Status >= 500 || s.Status == 401 || s.Status == 403 {
			sig += 2
		}
		if s.ApiCount > 0 {
			sig++
		}
		if s.UrlCount > 5 {
			sig++
		}
		if sig > 0 {
			scored = append(scored, rootScore{rec: s, score: sig})
		}
	}
	if len(scored) == 0 {
		for _, s := range data.Summary {
			scored = append(scored, rootScore{rec: s, score: s.ApiCount})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].rec.ApiCount > scored[j].rec.ApiCount
		}
		return scored[i].score > scored[j].score
	})

	maxRoots := 30
	if len(scored) < maxRoots {
		maxRoots = len(scored)
	}

	for idx := 0; idx < maxRoots; idx++ {
		s := scored[idx].rec
		line := fmt.Sprintf("ROOT %s | status=%d api=%d urls=%d cdn_urls=%d cdn_hosts=%s", s.Url, s.Status, s.ApiCount, s.UrlCount, s.CDNCount, truncate(compactText(s.CDNHosts), 80))
		if !writeLine(line) {
			break
		}

		bodies := bodiesByRoot[s.Url]
		if home := homepageSnippet(s.Url, bodies); home != "" {
			if title := extractTitle(home); title != "" {
				writeLine("Title: " + title)
			}
			writeLine("Snippet: " + home)
		}

		if kw := extractHomepageKeywords(bodies); len(kw) > 0 {
			writeLine("Keywords: " + strings.Join(kw, ", "))
		}

		if mc := mapsByRoot[s.Url]; mc > 0 {
			writeLine(fmt.Sprintf("SourceMaps: %d found", mc))
		}

		if sens := sensByRoot[s.Url]; len(sens) > 0 {
			writeLine("Sensitive:")
			sort.Slice(sens, func(i, j int) bool { return sens[i].Entropy > sens[j].Entropy })
			limit := 3
			if len(sens) < limit {
				limit = len(sens)
			}
			for i := 0; i < limit; i++ {
				hit := sens[i]
				writeLine(fmt.Sprintf("- %s ent=%.2f content=%s @%s", truncate(compactText(hit.Category), 40), hit.Entropy, truncate(compactText(hit.Content), 140), truncate(compactText(hit.SourceURL), 120)))
			}
		}

		if imp := impByRoot[s.Url]; len(imp) > 0 {
			writeLine("Important APIs:")
			limit := 3
			if len(imp) < limit {
				limit = len(imp)
			}
			for i := 0; i < limit; i++ {
				a := imp[i]
				writeLine(fmt.Sprintf("- %s (src %s)", truncate(compactText(a.Path), 120), truncate(compactText(a.SourceURL), 80)))
			}
		}

		if apis := apiByRoot[s.Url]; len(apis) > 0 && len(impByRoot[s.Url]) == 0 {
			writeLine("APIs (sample):")
			limit := 2
			if len(apis) < limit {
				limit = len(apis)
			}
			for i := 0; i < limit; i++ {
				a := apis[i]
				writeLine(fmt.Sprintf("- %s (src %s)", truncate(compactText(a.Path), 120), truncate(compactText(a.SourceURL), 80)))
			}
		}
		writeLine("")
	}

	return sb.String()
}

func compactText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func previewText(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + fmt.Sprintf("\n... (total %d chars)", len(s))
}

func rootOr(root string, src string) string {
	if root != "" {
		return root
	}
	return rootOf(src)
}

func homepageSnippet(root string, pages []PageSnapshotLite) string {
	if root == "" || len(pages) == 0 {
		return ""
	}
	var best PageSnapshotLite
	bestLen := 0
	for _, p := range pages {
		if rootOf(p.URL) != root && p.RootURL != root {
			continue
		}
		if p.Length > bestLen {
			best = p
			bestLen = p.Length
		}
	}
	if bestLen == 0 {
		return ""
	}
	// Compress: strip HTML tags to save tokens but keep content
	text := stripHTML(best.Snippet)
	if text == "" {
		text = compactText(best.Headers)
	}
	return truncate(compactText(text), 1200)
}

func stripHTML(html string) string {
	// Remove script/style blocks (handle truncated content with $)
	reScript := regexp.MustCompile(`(?si)<script[^>]*>.*?(?:</script>|$)|(?si)<style[^>]*>.*?(?:</style>|$)`)
	html = reScript.ReplaceAllString(html, " ")
	// Remove tags
	reTag := regexp.MustCompile(`<[^>]+>`)
	html = reTag.ReplaceAllString(html, " ")
	// Remove truncated tag at the end (e.g. "<div c")
	reTruncatedTag := regexp.MustCompile(`<[^>]*$`)
	html = reTruncatedTag.ReplaceAllString(html, " ")
	// Compact whitespace
	return compactText(html)
}

var wordSplit = regexp.MustCompile(`[A-Za-z0-9_]+`)

func extractHomepageKeywords(pages []PageSnapshotLite) []string {
	if len(pages) == 0 {
		return nil
	}
	type scored struct {
		word string
		cnt  int
	}
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true, "that": true, "this": true,
		"have": true, "your": true, "you": true, "are": true, "was": true, "will": true, "not": true,
		"can": true, "our": true, "all": true, "but": true, "has": true, "get": true, "use": true,
		"into": true, "about": true, "more": true, "info": true, "data": true, "user": true, "api": true,
		"http": true, "https": true, "www": true, "json": true, "html": true, "body": true, "href": true,
	}
	var buf strings.Builder
	for i, p := range pages {
		if i >= 6 { // top few bodies only
			break
		}
		buf.WriteString(p.Snippet)
		buf.WriteByte(' ')
		buf.WriteString(p.Headers)
		buf.WriteByte(' ')
	}
	text := strings.ToLower(buf.String())
	parts := wordSplit.FindAllString(text, -1)
	counts := make(map[string]int)
	for _, w := range parts {
		if len(w) < 4 {
			continue
		}
		if stop[w] {
			continue
		}
		counts[w]++
	}
	var items []scored
	for k, v := range counts {
		items = append(items, scored{word: k, cnt: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].cnt == items[j].cnt {
			return items[i].word < items[j].word
		}
		return items[i].cnt > items[j].cnt
	})
	limit := 15
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, items[i].word)
	}
	return out
}

func extractTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return compactText(m[1])
	}
	return ""
}

func callGoogleLLM(ctx context.Context, cfg *LLMConfig, input string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("llm config missing")
	}
	if strings.ToLower(cfg.Provider) != "google" {
		return "", fmt.Errorf("llm provider %s not supported", cfg.Provider)
	}
	if cfg.APIKey == "" {
		return "", fmt.Errorf("llm api key is empty")
	}
	model := cfg.Model
	if model == "" {
		model = DefaultLLMModel
	}
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = DefaultLLMPrompt
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": fmt.Sprintf("%s\n\n%s", prompt, input)},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": maxOutputTokens(cfg),
			"topK":            32,
			"topP":            0.95,
			"candidateCount":  1,
		},
	}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", url.PathEscape(model), url.QueryEscape(cfg.APIKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, truncate(compactText(string(respBody)), 240))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	for _, c := range parsed.Candidates {
		for _, p := range c.Content.Parts {
			if strings.TrimSpace(p.Text) != "" {
				return p.Text, nil
			}
		}
	}
	return "", fmt.Errorf("llm response empty")
}

func callOpenAI(ctx context.Context, cfg *LLMConfig, input string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("llm config missing")
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// If the user didn't include /v1 or /chat/completions, we need to be careful.
	// Standard OpenAI client takes a base URL that usually ends in /v1.
	// We will append /chat/completions.
	endpoint := baseURL + "/chat/completions"

	model := cfg.Model
	if model == "" {
		model = "gpt-3.5-turbo" // Fallback if no model specified for OpenAI
	}
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = DefaultLLMPrompt
	}

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": prompt},
			{"role": "user", "content": input},
		},
		"temperature": 0.3,
		"max_tokens":  maxOutputTokens(cfg),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai http %d: %s", resp.StatusCode, truncate(compactText(string(respBody)), 240))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) > 0 {
		return parsed.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("openai response empty")
}

func maxOutputTokens(cfg *LLMConfig) int {
	if cfg != nil && cfg.MaxOutputTok > 0 {
		return cfg.MaxOutputTok
	}
	// Default higher than previous 512 to reduce early truncation.
	return 10240
}
