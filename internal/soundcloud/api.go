package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type apiClient struct {
	credentials apiCredentials
	http        *http.Client
}

func newAPIClient(harFile string) (*apiClient, error) {
	credentials, err := loadCredentials(harFile)
	if err != nil {
		return nil, err
	}
	return &apiClient{credentials: credentials, http: &http.Client{}}, nil
}

func (c *Client) Mixes(ctx context.Context) ([]Track, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	var response mixedSelectionsResponse
	if err := c.api.get(ctx, "/mixed-selections", url.Values{
		"limit": {strconv.Itoa(c.limit)}, "linked_partitioning": {"1"},
	}, &response); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var tracks []Track
	for _, selection := range response.Collection {
		for _, playlist := range selection.Items.Collection {
			if playlist.Title == "" || !isSoundCloudURL(playlist.PermalinkURL) || seen[playlist.PermalinkURL] {
				continue
			}
			seen[playlist.PermalinkURL] = true
			ids := make([]int64, 0, len(playlist.Tracks))
			for _, track := range playlist.Tracks {
				if track.ID > 0 {
					ids = append(ids, int64(track.ID))
				}
			}
			tracks = append(tracks, Track{
				Title: playlist.Title, Artist: "SoundCloud", URL: playlist.PermalinkURL,
				Collection: true, TrackIDs: ids, Personalized: true,
			})
			if len(tracks) >= c.limit {
				return tracks, nil
			}
		}
	}
	if len(tracks) == 0 {
		return nil, errors.New("SoundCloud did not return any personal mixes")
	}
	return tracks, nil
}

func (c *Client) Likes(ctx context.Context) ([]Track, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	return c.api.trackCollection(ctx, "/users/"+url.PathEscape(c.api.credentials.userID)+"/track_likes", c.limit)
}

func (c *Client) History(ctx context.Context) ([]Track, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	return c.api.trackCollection(ctx, "/me/play-history/tracks", c.limit)
}

func (c *Client) requireAPI() error {
	c.apiOnce.Do(func() {
		c.api, c.apiErr = newAPIClient(c.harFile)
	})
	if c.api != nil {
		return nil
	}
	return fmt.Errorf("personal SoundCloud sections are unavailable: %w", c.apiErr)
}

func (a *apiClient) get(ctx context.Context, path string, query url.Values, target any) error {
	query.Set("client_id", a.credentials.clientID)
	if a.credentials.appVersion != "" {
		query.Set("app_version", a.credentials.appVersion)
	}
	if a.credentials.locale != "" {
		query.Set("app_locale", a.credentials.locale)
	}
	endpoint := "https://api-v2.soundcloud.com" + path + "?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", a.credentials.authorization)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Origin", "https://soundcloud.com")

	response, err := a.http.Do(request)
	if err != nil {
		return fmt.Errorf("SoundCloud API request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("SoundCloud API returned %s; refresh the HAR session", response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode SoundCloud API response: %w", err)
	}
	return nil
}

func (a *apiClient) trackCollection(ctx context.Context, path string, limit int) ([]Track, error) {
	var response trackCollectionResponse
	if err := a.get(ctx, path, url.Values{
		"limit": {strconv.Itoa(limit)}, "linked_partitioning": {"1"},
	}, &response); err != nil {
		return nil, err
	}
	apiTracks := make([]apiTrack, 0, len(response.Collection))
	for _, item := range response.Collection {
		apiTracks = append(apiTracks, item.Track)
	}
	return convertAPITracks(apiTracks)
}

func (a *apiClient) tracksByIDs(ctx context.Context, ids []int64) ([]Track, error) {
	if len(ids) == 0 {
		return nil, errors.New("персональный микс не содержит треков; обновите HAR-сессию")
	}
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.FormatInt(id, 10))
	}
	var response []apiTrack
	if err := a.get(ctx, "/tracks", url.Values{"ids": {strings.Join(values, ",")}}, &response); err != nil {
		return nil, err
	}
	return convertAPITracks(response)
}

func convertAPITracks(apiTracks []apiTrack) ([]Track, error) {
	tracks := make([]Track, 0, len(apiTracks))
	for _, track := range apiTracks {
		if track.Title == "" || !isSoundCloudURL(track.PermalinkURL) {
			continue
		}
		artist := track.User.Username
		if artist == "" {
			artist = "Unknown artist"
		}
		tracks = append(tracks, Track{
			Title: track.Title, Artist: artist, URL: track.PermalinkURL,
			Duration: track.Duration / 1000, Plays: track.PlaybackCount,
		})
	}
	if len(tracks) == 0 {
		return nil, errors.New("SoundCloud returned an empty track list")
	}
	return tracks, nil
}

type mixedSelectionsResponse struct {
	Collection []struct {
		Items struct {
			Collection []struct {
				Title        string `json:"title"`
				PermalinkURL string `json:"permalink_url"`
				Tracks       []struct {
					ID jsonID `json:"id"`
				} `json:"tracks"`
			} `json:"collection"`
		} `json:"items"`
	} `json:"collection"`
}

type trackCollectionResponse struct {
	Collection []struct {
		Track apiTrack `json:"track"`
	} `json:"collection"`
}

type apiTrack struct {
	ID            jsonID `json:"id"`
	Title         string `json:"title"`
	PermalinkURL  string `json:"permalink_url"`
	Duration      int    `json:"duration"`
	PlaybackCount int64  `json:"playback_count"`
	User          struct {
		Username string `json:"username"`
	} `json:"user"`
}

// jsonID accepts both numeric and quoted IDs used by different SoundCloud endpoints.
type jsonID int64

func (id *jsonID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*id = 0
		return nil
	}
	value := string(data)
	if data[0] == '"' {
		var quoted string
		if err := json.Unmarshal(data, &quoted); err != nil {
			return err
		}
		value = quoted
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid SoundCloud ID %q: %w", value, err)
	}
	*id = jsonID(parsed)
	return nil
}
