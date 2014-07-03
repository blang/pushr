package pushr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestLatestRelease(t *testing.T) {
	exampleRelease := Release{
		Name:    "Release1",
		Version: "1.0.0",
		Assets: []*Asset{
			{
				ID:          "1",
				Name:        "asset.bin",
				ContentType: "application/octet-stream",
			},
		},
	}
	var requestedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		if token := r.Header.Get("X-PUSHR-TOKEN"); token != "TOKEN123" {
			t.Errorf("Request with wrong token: %q", token)
			return
		}
		if requestedURL == "/repos/ns/repo/releases/stable/latest" {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(&exampleRelease)
			if err != nil {
				panic("Could not encode example")
			}
		} else if requestedURL == "/repos/ns/repo/assets/1" {
			w.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(w, "TESTOUTPUT")
		} else {
			t.Errorf("Wrong url requested: %q", requestedURL)
		}
	}))
	defer ts.Close()

	// Get Latest Release
	c := NewClient(ts.URL, "ns/repo", "TOKEN123")
	r, err := c.LatestRelease("stable")
	if err != nil {
		t.Fatalf("Error while getting latest release stable: %s", err)
	}
	if r == nil {
		t.Fatal("No Release found")
	}
	if !reflect.DeepEqual(r, &exampleRelease) {
		t.Fatalf("Release deep equal failed: expected %q, got %q", exampleRelease, r)
	}

	// Setup asset download
	tmpFile, err := ioutil.TempFile("", "pushrtest")
	if err != nil {
		t.Fatalf("Could not create test temp file: %s", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close() // Close, client will open it

	// Download fake asset
	err = c.Download(r.Assets[0], tmpFile.Name())
	if err != nil {
		t.Fatalf("Error while downloading asset: %s", err)
	}

	// Verify written tmp file
	b, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Could not read test temp file: %s", err)
	}
	if string(b) != "TESTOUTPUT" {
		t.Errorf("Written asset file contains wrong data: %q", string(b))
	}
}
