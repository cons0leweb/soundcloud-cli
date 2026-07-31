package soundcloud

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type harArchive struct {
	Log struct {
		Entries []struct {
			Request struct {
				Method      string    `json:"method"`
				URL         string    `json:"url"`
				Headers     []harPair `json:"headers"`
				QueryString []harPair `json:"queryString"`
			} `json:"request"`
		} `json:"entries"`
	} `json:"log"`
}

type harPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type apiCredentials struct {
	clientID      string
	authorization string
	appVersion    string
	locale        string
	userID        string
}

func loadCredentials(explicitPath string) (apiCredentials, error) {
	path, err := findHAR(explicitPath)
	if err != nil {
		return apiCredentials{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return apiCredentials{}, err
	}
	defer file.Close()

	var archive harArchive
	if err := json.NewDecoder(file).Decode(&archive); err != nil {
		return apiCredentials{}, err
	}

	var credentials apiCredentials
	for _, entry := range archive.Log.Entries {
		if entry.Request.Method != "GET" || !strings.HasPrefix(entry.Request.URL, "https://api-v2.soundcloud.com/") {
			continue
		}
		for _, header := range entry.Request.Headers {
			if strings.EqualFold(header.Name, "authorization") && header.Value != "" {
				credentials.authorization = header.Value
			}
		}
		for _, query := range entry.Request.QueryString {
			switch query.Name {
			case "client_id":
				credentials.clientID = query.Value
			case "app_version":
				credentials.appVersion = query.Value
			case "app_locale":
				credentials.locale = query.Value
			}
		}
		path := strings.Split(entry.Request.URL, "?")[0]
		const usersPrefix = "https://api-v2.soundcloud.com/users/"
		if strings.HasPrefix(path, usersPrefix) && strings.HasSuffix(path, "/track_likes") {
			credentials.userID = strings.TrimSuffix(strings.TrimPrefix(path, usersPrefix), "/track_likes")
		}
	}
	if credentials.clientID == "" || credentials.authorization == "" {
		return apiCredentials{}, errors.New("HAR does not contain an authenticated SoundCloud API request")
	}
	if credentials.userID == "" {
		return apiCredentials{}, errors.New("HAR does not contain the current SoundCloud user ID")
	}
	return credentials, nil
}

func findHAR(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("personal sections require --har /path/to/soundcloud.har")
	}
	var matches []string
	for _, directory := range []string{"Downloads", "Загрузки"} {
		found, _ := filepath.Glob(filepath.Join(home, directory, "soundcloud*.har"))
		matches = append(matches, found...)
	}
	if len(matches) == 0 {
		return "", errors.New("personal sections require --har /path/to/soundcloud.har")
	}
	sort.Slice(matches, func(i, j int) bool {
		left, leftErr := os.Stat(matches[i])
		right, rightErr := os.Stat(matches[j])
		if leftErr != nil || rightErr != nil {
			return matches[i] > matches[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	return matches[0], nil
}
