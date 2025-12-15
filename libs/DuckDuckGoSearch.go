package libs

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func SearchInternet(query string) string {
	searchURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible)")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	var results []string
	// DuckDuckGo structure can change — keep selector conservative.
	// Collect up to 5 results: title + snippet
	doc.Find(".result__body").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i >= 5 {
			return false
		}
		title := strings.TrimSpace(s.Find(".result__a").Text())
		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())
		if title == "" && snippet == "" {
			return true // continue
		}
		results = append(results, fmt.Sprintf("- %s: %s", title, snippet))
		return true
	})

	if len(results) == 0 {
		return ""
	}
	// join and cap length to avoid huge tokens
	out := strings.Join(results, "\n")
	if len(out) > 2000 {
		out = out[:2000] + "..."
	}
	return out
}
