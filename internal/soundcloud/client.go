package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Track is the small, UI-facing subset of SoundCloud metadata.
type Track struct {
	Title        string
	Artist       string
	URL          string
	Duration     int
	Plays        int64
	Collection   bool
	TrackIDs     []int64
	Personalized bool
}

// Expand returns tracks from either an authenticated system mix or a public set.
func (c *Client) Expand(ctx context.Context, collection Track) ([]Track, error) {
	if collection.Personalized {
		if len(collection.TrackIDs) == 0 {
			return nil, errors.New("SoundCloud не передал треки персонального микса; обновите HAR-сессию")
		}
		if err := c.requireAPI(); err != nil {
			return nil, err
		}
		return c.api.tracksByIDs(ctx, collection.TrackIDs)
	}
	return c.Search(ctx, collection.URL)
}

// Client uses yt-dlp as the compatibility layer for SoundCloud's changing API.
type Client struct {
	binary  string
	cookies string
	limit   int
	harFile string
	apiOnce sync.Once
	api     *apiClient
	apiErr  error
}

func New(binary, cookies, harFile string, limit int) *Client {
	return &Client{binary: binary, cookies: cookies, harFile: harFile, limit: limit}
}

func (c *Client) Search(ctx context.Context, query string) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("enter an artist, track, or genre")
	}

	target, err := c.searchTarget(query)
	if err != nil {
		return nil, err
	}
	args := c.withCookies(
		"--dump-single-json",
		"--flat-playlist",
		"--playlist-end", strconv.Itoa(c.limit),
		"--no-warnings",
		"--no-call-home",
		target,
	)
	cmd := exec.CommandContext(ctx, c.binary, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError("SoundCloud search failed", err)
	}

	tracks, err := parseSearch(output)
	if err != nil {
		return nil, fmt.Errorf("decode SoundCloud results: %w", err)
	}
	if len(tracks) == 0 {
		return nil, errors.New("nothing found; try a broader query")
	}
	return tracks, nil
}

func (c *Client) searchTarget(query string) (string, error) {
	if isSoundCloudURL(query) {
		return query, nil
	}
	if !strings.HasPrefix(query, "@") {
		return "scsearch" + strconv.Itoa(c.limit) + ":" + query, nil
	}

	profile := strings.TrimPrefix(query, "@")
	suffix := ""
	for _, candidate := range []string{"/sets", "/likes"} {
		if strings.HasSuffix(profile, candidate) {
			profile = strings.TrimSuffix(profile, candidate)
			suffix = candidate
			break
		}
	}
	if profile == "" || strings.ContainsAny(profile, "/?# ") {
		return "", errors.New("use @username, @username/sets, or @username/likes")
	}
	return "https://soundcloud.com/" + url.PathEscape(profile) + suffix, nil
}

// StreamURL resolves a public SoundCloud page to the media URL ffplay can read.
func (c *Client) StreamURL(ctx context.Context, pageURL string) (string, error) {
	if !isSoundCloudURL(pageURL) {
		return "", errors.New("refusing to play a non-SoundCloud URL")
	}

	args := c.withCookies(
		"--get-url",
		"--format", "bestaudio/best",
		"--no-playlist",
		"--no-warnings",
		"--no-call-home",
		pageURL,
	)
	cmd := exec.CommandContext(ctx, c.binary, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", commandError("stream resolution failed", err)
	}
	streamURL := strings.TrimSpace(strings.Split(string(output), "\n")[0])
	if streamURL == "" {
		return "", errors.New("SoundCloud returned an empty stream URL")
	}
	return streamURL, nil
}

func (c *Client) withCookies(args ...string) []string {
	if strings.TrimSpace(c.cookies) == "" {
		return args
	}
	return append([]string{"--cookies", c.cookies}, args...)
}

type searchResponse struct {
	searchEntry
	Entries []searchEntry `json:"entries"`
}

type searchEntry struct {
	Title      string  `json:"title"`
	Uploader   string  `json:"uploader"`
	URL        string  `json:"url"`
	WebpageURL string  `json:"webpage_url"`
	Duration   float64 `json:"duration"`
	ViewCount  int64   `json:"view_count"`
}

func parseSearch(data []byte) ([]Track, error) {
	var response searchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	tracks := make([]Track, 0, len(response.Entries))
	if len(response.Entries) == 0 && response.Title != "" {
		response.Entries = append(response.Entries, response.searchEntry)
	}
	for _, entry := range response.Entries {
		pageURL := entry.WebpageURL
		if pageURL == "" && isSoundCloudURL(entry.URL) {
			pageURL = entry.URL
		}
		if entry.Title == "" || !isSoundCloudURL(pageURL) {
			continue
		}
		artist := strings.TrimSpace(entry.Uploader)
		if artist == "" {
			artist = "Unknown artist"
		}
		tracks = append(tracks, Track{
			Title:      strings.TrimSpace(entry.Title),
			Artist:     artist,
			URL:        pageURL,
			Duration:   int(entry.Duration + 0.5),
			Plays:      entry.ViewCount,
			Collection: isCollectionURL(pageURL),
		})
	}
	return tracks, nil
}

func isCollectionURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || !isSoundCloudURL(raw) {
		return false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	return len(parts) >= 3 && parts[1] == "sets"
}

func isSoundCloudURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return parsed.Scheme == "https" && (host == "soundcloud.com" || strings.HasSuffix(host, ".soundcloud.com"))
}

func commandError(prefix string, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		detail := strings.TrimSpace(string(exitErr.Stderr))
		if detail != "" {
			return fmt.Errorf("%s: %s", prefix, detail)
		}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
