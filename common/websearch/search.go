package websearch

import (
	appconfig "GopherAI/config"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"
)

type Focus string

const (
	FocusAuto    Focus = "auto"
	FocusGeneral Focus = "general"
	FocusNews    Focus = "news"
)

type Result struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet,omitempty"`
	Source      string `json:"source"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

type Response struct {
	Query   string   `json:"query"`
	Focus   Focus    `json:"focus"`
	Results []Result `json:"results"`
}

type wikipediaResponse struct {
	Query struct {
		Search []struct {
			Title     string `json:"title"`
			Snippet   string `json:"snippet"`
			Timestamp string `json:"timestamp"`
		} `json:"search"`
	} `json:"query"`
}

type googleNewsRSS struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

type hackerNewsResponse struct {
	Hits []struct {
		Title      string `json:"title"`
		StoryTitle string `json:"story_title"`
		URL        string `json:"url"`
		StoryURL   string `json:"story_url"`
		StoryText  string `json:"story_text"`
		CreatedAt  string `json:"created_at"`
		ObjectID   string `json:"objectID"`
	} `json:"hits"`
}

var (
	htmlTagRegex  = regexp.MustCompile(`<[^>]+>`)
	asciiTokenReg = regexp.MustCompile(`[A-Za-z][A-Za-z0-9._:+-]*`)
	hanTokenReg   = regexp.MustCompile(`[\p{Han}]{2,}`)
	punctRegex    = regexp.MustCompile(`[？?，,。.!！:：、；;（）()【】\[\]“”"']+`)
)

func Search(ctx context.Context, query string, focus Focus, maxResults int) (*Response, error) {
	conf := appconfig.GetConfig()
	if !conf.SearchConfig.Enabled {
		return nil, fmt.Errorf("web search is disabled")
	}

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	if maxResults <= 0 {
		maxResults = conf.SearchConfig.MaxResults
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	resolvedFocus := normalizeFocus(query, focus)
	searchQuery := buildSearchQuery(query, resolvedFocus)
	results := make([]Result, 0, maxResults)

	switch resolvedFocus {
	case FocusNews:
		newsResults, _ := searchGoogleNews(ctx, searchQuery, maxResults)
		results = append(results, newsResults...)
		if len(results) < maxResults {
			hnResults, _ := searchHackerNews(ctx, searchQuery, maxResults-len(results))
			results = append(results, hnResults...)
		}
	default:
		generalResults, _ := searchWikipedia(ctx, searchQuery, maxResults)
		results = append(results, generalResults...)

		if len(results) < maxResults {
			newsResults, _ := searchGoogleNews(ctx, searchQuery, maxResults-len(results))
			results = append(results, newsResults...)
		}
		if len(results) < maxResults {
			hnResults, _ := searchHackerNews(ctx, searchQuery, maxResults-len(results))
			results = append(results, hnResults...)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no web search results found")
	}

	return &Response{
		Query:   query,
		Focus:   resolvedFocus,
		Results: dedupeResults(results, maxResults),
	}, nil
}

func FormatToolResult(resp *Response) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("联网搜索结果（query=%s, focus=%s）:\n", resp.Query, resp.Focus))
	for i, result := range resp.Results {
		builder.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, result.Source, result.Title))
		builder.WriteString(fmt.Sprintf("URL: %s\n", result.URL))
		if result.PublishedAt != "" {
			builder.WriteString(fmt.Sprintf("发布时间: %s\n", result.PublishedAt))
		}
		if result.Snippet != "" {
			builder.WriteString(fmt.Sprintf("摘要: %s\n", result.Snippet))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("请基于以上结果作答；如果结果不足以支撑结论，请明确说明。")
	return builder.String()
}

func SearchJSON(ctx context.Context, query string, focus Focus, maxResults int) (string, error) {
	resp, err := Search(ctx, query, focus, maxResults)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func normalizeFocus(query string, focus Focus) Focus {
	switch focus {
	case FocusNews, FocusGeneral:
		return focus
	default:
		if looksTimeSensitive(query) {
			return FocusNews
		}
		return FocusGeneral
	}
}

func looksTimeSensitive(query string) bool {
	lower := strings.ToLower(query)
	keywords := []string{
		"今天", "最新", "最近", "刚刚", "当前", "现在", "新闻",
		"today", "latest", "recent", "news", "current", "now",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func searchWikipedia(ctx context.Context, query string, maxResults int) ([]Result, error) {
	base := "https://en.wikipedia.org/w/api.php"
	wikiHost := "https://en.wikipedia.org/wiki/"
	if containsChinese(query) {
		base = "https://zh.wikipedia.org/w/api.php"
		wikiHost = "https://zh.wikipedia.org/wiki/"
	}

	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", query)
	params.Set("utf8", "1")
	params.Set("format", "json")
	params.Set("srlimit", fmt.Sprintf("%d", maxResults))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := defaultHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikipedia search returned status %d", resp.StatusCode)
	}

	var payload wikipediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(payload.Query.Search))
	for _, item := range payload.Query.Search {
		results = append(results, Result{
			Title:       item.Title,
			URL:         wikiHost + url.PathEscape(strings.ReplaceAll(item.Title, " ", "_")),
			Snippet:     cleanSnippet(item.Snippet),
			Source:      "wikipedia",
			PublishedAt: item.Timestamp,
		})
	}
	return results, nil
}

func searchGoogleNews(ctx context.Context, query string, maxResults int) ([]Result, error) {
	conf := appconfig.GetConfig()
	params := url.Values{}
	params.Set("q", query)
	params.Set("hl", conf.SearchConfig.NewsLanguage)
	params.Set("gl", conf.SearchConfig.NewsRegion)
	params.Set("ceid", conf.SearchConfig.NewsEdition)

	body, err := fetchGoogleNewsRSS(ctx, "https://news.google.com/rss/search?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var rss googleNewsRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, err
	}

	results := make([]Result, 0, maxResults)
	for _, item := range rss.Channel.Items {
		results = append(results, Result{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.Link),
			Snippet:     cleanSnippet(item.Description),
			Source:      "google-news",
			PublishedAt: strings.TrimSpace(item.PubDate),
		})
		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

func fetchGoogleNewsRSS(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err == nil {
		resp, doErr := defaultHTTPClient().Do(req)
		if doErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, readErr := io.ReadAll(resp.Body)
				if readErr == nil && len(bytes.TrimSpace(body)) > 0 {
					return body, nil
				}
				if readErr != nil {
					err = readErr
				}
			} else {
				err = fmt.Errorf("google news search returned status %d", resp.StatusCode)
			}
		} else {
			err = doErr
		}
	}

	if runtime.GOOS == "windows" {
		body, psErr := fetchURLViaPowerShell(ctx, endpoint)
		if psErr == nil && len(bytes.TrimSpace(body)) > 0 {
			return body, nil
		}
		if err != nil && psErr != nil {
			return nil, fmt.Errorf("%v; powershell fallback failed: %w", err, psErr)
		}
		if psErr != nil {
			return nil, psErr
		}
	}

	return nil, err
}

func fetchURLViaPowerShell(ctx context.Context, endpoint string) ([]byte, error) {
	script := fmt.Sprintf(
		"$ProgressPreference='SilentlyContinue'; [Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); $resp = Invoke-WebRequest -UseBasicParsing -Uri '%s'; [Console]::Out.Write($resp.Content)",
		escapePowerShellSingleQuoted(endpoint),
	)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}

	return output, nil
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func searchHackerNews(ctx context.Context, query string, maxResults int) ([]Result, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("tags", "story")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://hn.algolia.com/api/v1/search_by_date?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := defaultHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hacker news search returned status %d", resp.StatusCode)
	}

	var payload hackerNewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]Result, 0, maxResults)
	for _, hit := range payload.Hits {
		title := strings.TrimSpace(hit.Title)
		if title == "" {
			title = strings.TrimSpace(hit.StoryTitle)
		}

		link := strings.TrimSpace(hit.URL)
		if link == "" {
			link = strings.TrimSpace(hit.StoryURL)
		}
		if link == "" && hit.ObjectID != "" {
			link = "https://news.ycombinator.com/item?id=" + hit.ObjectID
		}

		if title == "" || link == "" {
			continue
		}

		results = append(results, Result{
			Title:       title,
			URL:         link,
			Snippet:     cleanSnippet(hit.StoryText),
			Source:      "hacker-news",
			PublishedAt: strings.TrimSpace(hit.CreatedAt),
		})
		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

func buildSearchQuery(query string, focus Focus) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	normalized := normalizeSearchQuery(query)
	if normalized == "" {
		normalized = query
	}

	asciiTokens := filterSearchTokens(asciiTokenReg.FindAllString(normalized, -1))
	if focus == FocusNews && len(asciiTokens) > 0 {
		return strings.Join(dedupeStrings(asciiTokens, 4), " ")
	}

	hanTokens := filterSearchTokens(hanTokenReg.FindAllString(normalized, -1))
	combined := append([]string{}, asciiTokens...)
	combined = append(combined, hanTokens...)
	combined = dedupeStrings(combined, 6)
	if len(combined) == 0 {
		return normalized
	}

	return strings.Join(combined, " ")
}

func normalizeSearchQuery(query string) string {
	query = punctRegex.ReplaceAllString(query, " ")

	replacements := []string{
		"最近怎么了", " ",
		"最近发生了什么", " ",
		"最近有啥事", " ",
		"最近有什么事", " ",
		"最近", " ",
		"最新", " ",
		"新闻", " ",
		"热点", " ",
		"怎么了", " ",
		"发生了什么", " ",
		"请列出", " ",
		"列出", " ",
		"并注明来源", " ",
		"注明来源", " ",
		"来源", " ",
		"请问", " ",
		"一下", " ",
	}
	query = strings.NewReplacer(replacements...).Replace(query)
	return strings.Join(strings.Fields(query), " ")
}

func filterSearchTokens(tokens []string) []string {
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" || shouldSkipSearchToken(token) {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

func shouldSkipSearchToken(token string) bool {
	lower := strings.ToLower(token)
	englishStopWords := map[string]struct{}{
		"latest": {}, "recent": {}, "news": {}, "current": {}, "today": {},
		"please": {}, "list": {}, "show": {}, "source": {}, "sources": {},
		"what": {}, "whats": {}, "tell": {}, "about": {},
	}
	if _, exists := englishStopWords[lower]; exists {
		return true
	}

	chineseNoise := []string{"最近", "最新", "新闻", "什么", "请", "列出", "注明", "来源", "告诉我", "一下", "并"}
	if !containsASCII(token) {
		for _, noise := range chineseNoise {
			if strings.Contains(token, noise) {
				return true
			}
		}
	}

	return false
}

func dedupeStrings(values []string, limit int) []string {
	if limit <= 0 {
		limit = len(values)
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, limit)
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, strings.TrimSpace(value))
		if len(result) >= limit {
			break
		}
	}
	return result
}

func dedupeResults(results []Result, maxResults int) []Result {
	seen := make(map[string]struct{}, len(results))
	deduped := make([]Result, 0, len(results))
	for _, result := range results {
		key := result.URL
		if key == "" {
			key = result.Title
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, result)
		if len(deduped) >= maxResults {
			break
		}
	}
	return deduped
}

func cleanSnippet(value string) string {
	value = html.UnescapeString(value)
	value = htmlTagRegex.ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func containsChinese(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func containsASCII(value string) bool {
	for _, r := range value {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return true
		}
	}
	return false
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}
