package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiURL    = "https://youtubei.googleapis.com/youtubei/v1/search?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	UserAgent = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Mobile Safari/537.36"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type ClientContext struct {
	Client struct {
		ClientName    string `json:"clientName"`
		ClientVersion string `json:"clientVersion"`
	} `json:"client"`
}

var (
	defaultContext = ClientContext{
		Client: struct {
			ClientName    string `json:"clientName"`
			ClientVersion string `json:"clientVersion"`
		}{
			ClientName:    "ANDROID_LITE",
			ClientVersion: "3.29",
		},
	}
)

type Video struct {
	ID       string
	Title    string
	Channel  string
	Duration time.Duration
	Views    uint64
}

type SearchResponse struct {
	Contents struct {
		SectionListRenderer struct {
			Contents []struct {
				ItemSectionRenderer struct {
					Contents []struct {
						CompactVideoRenderer VideoRenderer `json:"compactVideoRenderer"`
					} `json:"contents"`
				} `json:"itemSectionRenderer"`
			} `json:"contents"`
		} `json:"sectionListRenderer"`
	} `json:"contents"`
}

type VideoRenderer struct {
	VideoID        string `json:"videoId"`
	Title          Text   `json:"title"`
	LongBylineText Text   `json:"longBylineText"`
	LengthText     Text   `json:"lengthText"`
	ViewCountText  Text   `json:"viewCountText"`
}

type Text struct {
	Runs []struct {
		Text string `json:"text"`
	} `json:"runs"`
}

func Search(query string) ([]Video, error) {

	body := map[string]interface{}{
		"query":   query,
		"context": defaultContext,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("making HTTP POST request: %w", err)
	}
	defer resp.Body.Close()

	var result SearchResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var videos []Video
	for _, section := range result.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			videoData := item.CompactVideoRenderer
			if videoData.VideoID == "" {
				continue
			}

			title := videoData.Title.Runs[0].Text

			channel := videoData.LongBylineText.Runs[0].Text

			duration, err := parseDuration(videoData.LengthText.Runs[0].Text)
			if err != nil {
				fmt.Println("Error parsing duration: ", err)
				continue
			}

			views, err := parseViews(videoData.ViewCountText.Runs[0].Text)
			if err != nil {
				fmt.Println("Error parsing views: ", err)
				continue
			}

			video := Video{
				Title:    title,
				ID:       videoData.VideoID,
				Channel:  channel,
				Duration: duration,
				Views:    views,
			}
			videos = append(videos, video)
		}
	}

	return videos, nil
}

func parseDuration(duration string) (time.Duration, error) {
	parts := strings.Split(duration, ":")
	var hours, minutes, seconds int
	var err error

	// Handle different possible formats
	if len(parts) == 2 { // MM:SS
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, errors.New("invalid minutes value")
		}
		seconds, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, errors.New("invalid seconds value")
		}
	} else if len(parts) == 3 { // H:MM:SS
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, errors.New("invalid hours value")
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, errors.New("invalid minutes value")
		}
		seconds, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, errors.New("invalid seconds value")
		}
	} else {
		return 0, errors.New("invalid duration format")
	}

	// Ensure valid time components
	if minutes >= 60 || seconds >= 60 {
		return 0, errors.New("minutes and seconds must be less than 60")
	}

	// Convert to time.Duration
	totalSeconds := hours*3600 + minutes*60 + seconds
	return time.Duration(totalSeconds) * time.Second, nil
}

func parseViews(views string) (uint64, error) {
	views = strings.ReplaceAll(strings.ReplaceAll(views, " views", ""), ",", "")
	return strconv.ParseUint(views, 10, 64)
}
