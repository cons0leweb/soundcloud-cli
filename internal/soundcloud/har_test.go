package soundcloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCredentialsDoesNotRequireLibraryUserID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.har")
	data := []byte(`{"log":{"entries":[{"request":{"method":"GET","url":"https://api-v2.soundcloud.com/me/play-history?client_id=fresh-client&app_version=1","headers":[{"name":"Authorization","value":"OAuth fresh-session"}],"queryString":[{"name":"client_id","value":"fresh-client"},{"name":"app_version","value":"1"}]}}]}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	credentials, err := loadCredentials(path)
	if err != nil {
		t.Fatalf("fresh HAR without user track_likes was rejected: %v", err)
	}
	if credentials.clientID != "fresh-client" || credentials.authorization == "" || credentials.userID != "" {
		t.Fatalf("unexpected credentials: client=%q auth=%t user=%q", credentials.clientID, credentials.authorization != "", credentials.userID)
	}
}
