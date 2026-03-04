package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type WebScraper struct {
	http     *http.Client
	maxBytes int64
}

func NewWebScraper() *WebScraper {
	return &WebScraper{
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxBytes: 512 * 1024, // 512KB limit per page
	}
}

func (ws *WebScraper) Scrape(pageURL string) (string, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRAGBot/1.0)")
	req.Header.Set("Accept", "text/html")

	resp, err := ws.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "text/plain") {
		return "", fmt.Errorf("unsupported content type: %s", ct)
	}

	limited := io.LimitReader(resp.Body, ws.maxBytes)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}

	return extractReadableText(string(raw)), nil
}

var (
	scriptRe     = regexp.MustCompile(`(?si)<script[^>]*>.*?</script>`)
	styleRe      = regexp.MustCompile(`(?si)<style[^>]*>.*?</style>`)
	noscriptRe   = regexp.MustCompile(`(?si)<noscript[^>]*>.*?</noscript>`)
	navRe        = regexp.MustCompile(`(?si)<nav[^>]*>.*?</nav>`)
	footerRe     = regexp.MustCompile(`(?si)<footer[^>]*>.*?</footer>`)
	headerRe     = regexp.MustCompile(`(?si)<header[^>]*>.*?</header>`)
	iframeRe     = regexp.MustCompile(`(?si)<iframe[^>]*>.*?</iframe>`)
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	allTagsRe     = regexp.MustCompile(`<[^>]+>`)
	entityRe      = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)
	multiSpaceRe  = regexp.MustCompile(`[ \t]+`)
	multiNewline  = regexp.MustCompile(`\n{3,}`)
)

func extractReadableText(html string) string {
	text := scriptRe.ReplaceAllString(html, " ")
	text = styleRe.ReplaceAllString(text, " ")
	text = noscriptRe.ReplaceAllString(text, " ")
	text = navRe.ReplaceAllString(text, " ")
	text = footerRe.ReplaceAllString(text, " ")
	text = headerRe.ReplaceAllString(text, " ")
	text = iframeRe.ReplaceAllString(text, " ")
	text = htmlCommentRe.ReplaceAllString(text, " ")

	// Turn block elements into newlines so paragraphs stay separated
	blockTags := regexp.MustCompile(`(?i)</(p|div|li|tr|h[1-6]|article|section|blockquote)>`)
	text = blockTags.ReplaceAllString(text, "\n")
	brTag := regexp.MustCompile(`(?i)<br\s*/?>`)
	text = brTag.ReplaceAllString(text, "\n")

	text = allTagsRe.ReplaceAllString(text, " ")
	text = entityRe.ReplaceAllStringFunc(text, decodeEntity)
	text = multiSpaceRe.ReplaceAllString(text, " ")
	text = multiNewline.ReplaceAllString(text, "\n\n")

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 20 {
			lines = append(lines, trimmed)
		}
	}
	return strings.Join(lines, "\n")
}

func decodeEntity(entity string) string {
	common := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&apos;": "'",
		"&#39;":  "'",
		"&nbsp;": " ",
		"&mdash;": "—",
		"&ndash;": "–",
		"&hellip;": "...",
	}
	if v, ok := common[entity]; ok {
		return v
	}
	return " "
}
