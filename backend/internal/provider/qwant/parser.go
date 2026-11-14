package qwant

import (
	"encoding/json"
	"strings"
	"time"

	"plewtus/internal/model"
)

// Response is a decoded Qwant API reply.
type Response struct {
	Results []model.RawSearchResult
	// ErrorCode is data.error_code when status is "error", else 0.
	ErrorCode int
	// Captcha is set when DataDome answered with a captcha redirect
	// (a top-level "url" instead of results).
	Captcha bool
}

type apiResponse struct {
	Status string  `json:"status"`
	URL    *string `json:"url"`
	Data   struct {
		ErrorCode int `json:"error_code"`
		Result    struct {
			Items struct {
				Mainline []struct {
					Type  string    `json:"type"`
					Items []apiItem `json:"items"`
				} `json:"mainline"`
			} `json:"items"`
		} `json:"result"`
	} `json:"data"`
}

type apiItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Desc  string `json:"desc"`
	Date  int64  `json:"date"`
}

// ParseResponse decodes a Qwant API body. Web results come from the
// data.result.items.mainline rows of type "web"; ads and other verticals
// (videos, news carousels) are skipped.
func ParseResponse(body []byte) (Response, error) {
	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Response{}, err
	}
	if parsed.URL != nil && *parsed.URL != "" {
		return Response{Captcha: true}, nil
	}
	if parsed.Status != "success" {
		code := parsed.Data.ErrorCode
		if code == 0 {
			code = -1 // an error without a code
		}
		return Response{ErrorCode: code}, nil
	}

	var out Response
	for _, row := range parsed.Data.Result.Items.Mainline {
		if row.Type != "web" {
			continue
		}
		for _, item := range row.Items {
			link := strings.TrimSpace(item.URL)
			if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
				continue
			}
			title := strings.Join(strings.Fields(item.Title), " ")
			if title == "" {
				title = link
			}
			out.Results = append(out.Results, model.RawSearchResult{
				Title:         title,
				URL:           link,
				Snippet:       strings.Join(strings.Fields(item.Desc), " "),
				Position:      len(out.Results) + 1,
				PublishedDate: formatDate(item.Date),
			})
		}
	}
	return out, nil
}

// formatDate renders Qwant's epoch date (seconds, occasionally milliseconds).
func formatDate(epoch int64) string {
	if epoch <= 0 {
		return ""
	}
	if epoch > 1e12 {
		epoch /= 1000
	}
	return time.Unix(epoch, 0).UTC().Format("2006-01-02")
}
