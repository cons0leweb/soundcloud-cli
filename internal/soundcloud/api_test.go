package soundcloud

import (
	"encoding/json"
	"testing"
)

func TestJSONIDAcceptsNumberAndString(t *testing.T) {
	for _, input := range []string{`12345`, `"12345"`} {
		var value struct {
			ID jsonID `json:"id"`
		}
		if err := json.Unmarshal([]byte(`{"id":`+input+`}`), &value); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		if value.ID != 12345 {
			t.Fatalf("unmarshal %s = %d, want 12345", input, value.ID)
		}
	}
}

func TestJSONIDRejectsInvalidValue(t *testing.T) {
	var value struct {
		ID jsonID `json:"id"`
	}
	if err := json.Unmarshal([]byte(`{"id":"not-an-id"}`), &value); err == nil {
		t.Fatal("invalid ID was accepted")
	}
}

func TestMixedSelectionAcceptsURNContainerID(t *testing.T) {
	data := []byte(`{
  "collection": [{
    "items": {"collection": [{
      "id": "soundcloud:system-playlists:your-moods:42:1",
      "title": "Your Mix 1",
      "permalink_url": "https://soundcloud.com/discover/sets/your-mix-1",
      "tracks": [{"id": "123"}, {"id": 456}]
    }]}
  }]
}`)
	var response mixedSelectionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	tracks := response.Collection[0].Items.Collection[0].Tracks
	if len(tracks) != 2 || tracks[0].ID != 123 || tracks[1].ID != 456 {
		t.Fatalf("unexpected track IDs: %#v", tracks)
	}
}

func TestPreferredTranscodingPrefersProgressive(t *testing.T) {
	transcodings := []apiTranscoding{{URL: "https://api-v2.soundcloud.com/hls"}, {URL: "https://api-v2.soundcloud.com/progressive"}}
	transcodings[0].Format.Protocol = "hls"
	transcodings[0].Format.MIMEType = "audio/mpeg"
	transcodings[1].Format.Protocol = "progressive"
	transcodings[1].Format.MIMEType = "audio/mpeg"
	if got := preferredTranscoding(transcodings); got != transcodings[1].URL {
		t.Fatalf("preferred transcoding = %q, want progressive %q", got, transcodings[1].URL)
	}
}
