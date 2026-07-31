package soundcloud

import (
	"slices"
	"testing"
)

func TestParseSearch(t *testing.T) {
	data := []byte(`{
  "entries": [
    {
      "title": "Night Drive",
      "uploader": "DJ Test",
      "webpage_url": "https://soundcloud.com/dj-test/night-drive",
      "duration": 183.6,
      "view_count": 12500
    },
    {
      "title": "Not SoundCloud",
      "webpage_url": "https://example.com/track"
    },
    {
      "title": "Fallback URL",
      "url": "https://soundcloud.com/test/fallback",
      "duration": 10
    }
  ]
}`)

	tracks, err := parseSearch(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Title != "Night Drive" || tracks[0].Artist != "DJ Test" {
		t.Fatalf("unexpected first track: %#v", tracks[0])
	}
	if tracks[0].Duration != 184 || tracks[0].Plays != 12500 {
		t.Fatalf("unexpected numeric metadata: %#v", tracks[0])
	}
	if tracks[1].Artist != "Unknown artist" {
		t.Fatalf("missing artist was not normalized: %#v", tracks[1])
	}
}

func TestParseSearchRejectsInvalidJSON(t *testing.T) {
	if _, err := parseSearch([]byte(`{"entries":`)); err == nil {
		t.Fatal("parseSearch accepted invalid JSON")
	}
}

func TestSoundCloudURLValidation(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://soundcloud.com/artist/track", true},
		{"https://m.soundcloud.com/artist/track", true},
		{"http://soundcloud.com/artist/track", false},
		{"https://soundcloud.com.example.org/artist/track", false},
		{"not a url", false},
	}
	for _, test := range tests {
		if got := isSoundCloudURL(test.url); got != test.want {
			t.Errorf("isSoundCloudURL(%q) = %v, want %v", test.url, got, test.want)
		}
	}
}

func TestCollectionURLValidation(t *testing.T) {
	if !isCollectionURL("https://soundcloud.com/artist/sets/night-mix") {
		t.Fatal("set URL was not recognized as a collection")
	}
	if isCollectionURL("https://soundcloud.com/artist/single") {
		t.Fatal("track URL was recognized as a collection")
	}
}

func TestSearchTarget(t *testing.T) {
	client := &Client{binary: "yt-dlp", limit: 20}
	tests := map[string]string{
		"deep house":                        "scsearch20:deep house",
		"@artist":                           "https://soundcloud.com/artist",
		"@artist/sets":                      "https://soundcloud.com/artist/sets",
		"@artist/likes":                     "https://soundcloud.com/artist/likes",
		"https://soundcloud.com/artist/mix": "https://soundcloud.com/artist/mix",
	}
	for query, want := range tests {
		got, err := client.searchTarget(query)
		if err != nil {
			t.Fatalf("searchTarget(%q): %v", query, err)
		}
		if got != want {
			t.Errorf("searchTarget(%q) = %q, want %q", query, got, want)
		}
	}
}

func TestYTDLPArgsDoNotUseRemovedOptions(t *testing.T) {
	client := &Client{binary: "yt-dlp", cookies: "cookies.txt", limit: 20}
	for name, args := range map[string][]string{
		"search": client.searchArgs("scsearch20:test"),
		"stream": client.streamArgs("https://soundcloud.com/user/track"),
	} {
		if slices.Contains(args, "--no-call-home") {
			t.Fatalf("%s args contain removed --no-call-home: %v", name, args)
		}
		if !slices.Contains(args, "--cookies") {
			t.Fatalf("%s args lost cookie support: %v", name, args)
		}
		if !slices.Contains(args, "--ignore-config") {
			t.Fatalf("%s args do not isolate deprecated user config: %v", name, args)
		}
	}
	publicArgs := client.streamArgsWithoutCookies("https://soundcloud.com/user/track")
	if slices.Contains(publicArgs, "--cookies") || slices.Contains(publicArgs, "cookies.txt") {
		t.Fatalf("cookie-free retry contains cookies: %v", publicArgs)
	}
}
