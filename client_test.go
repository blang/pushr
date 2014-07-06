package pushr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLatestRelease(t *testing.T) {
	testRelease := Release{
		Versions: map[string]*Version{
			"1.0.1-beta": &Version{
				ContentType: "application/zip",
				Size:        10,
				Filename:    "test-1.0.1-beta.zip",
			},
			"1.0.0": &Version{
				ContentType: "application/zip",
				Size:        10,
				Filename:    "test-1.0.0.zip",
			},
			"0.1.0": &Version{
				ContentType: "application/zip",
				Size:        10,
				Filename:    "test-0.1.0.zip",
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
		t.Logf("URL: %s", requestedURL)
		if requestedURL == "/releases/test" {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(&testRelease)
			if err != nil {
				panic("Could not encode example")
			}
		} else if requestedURL == "/releases/test/1.0.0" {
			if strings.Contains(r.Header.Get("Accept"), "application/json") {
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(testRelease.Versions["1.0.0"])
				if err != nil {
					panic("Could not encode example")
				}
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
				fmt.Fprint(w, "TESTOUTPUT")
			}
		} else {
			t.Errorf("Wrong url requested: %q", requestedURL)
		}
	}))
	defer ts.Close()

	// Get Release
	c := NewClient(ts.URL, "TOKEN123", "")
	r, err := c.Release("test")
	if err != nil {
		t.Fatalf("Error while getting releases: %s", err)
	}
	if r == nil {
		t.Fatal("No Release found")
	}
	if !reflect.DeepEqual(r, &testRelease) {
		t.Fatalf("Release deep equal failed: expected %q, got %q", testRelease, r)
	}

	// Get Version
	v, err := c.Version("test", "1.0.0")
	if err != nil {
		t.Fatalf("Error while getting version: %s", err)
	}
	if v == nil {
		t.Fatal("No version found")
	}
	if !reflect.DeepEqual(v, testRelease.Versions["1.0.0"]) {
		t.Fatalf("Version deep equal failed: expected %q, got %q", testRelease.Versions["1.0.0"], v)
	}

	// Setup asset download
	tmpFile, err := ioutil.TempFile("", "pushrtest")
	if err != nil {
		t.Fatalf("Could not create test temp file: %s", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close() // Close, client will open it

	// Download fake asset
	err = c.Download("test", "1.0.0", tmpFile.Name())
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

	//Get latest stable version
	v, versionStr, err := c.LatestVersion("test", "") //stable
	if !reflect.DeepEqual(v, testRelease.Versions["1.0.0"]) {
		t.Fatalf("Latest version on stable channel failed: expected %q, got %q", testRelease.Versions["1.0.0"], v)
	}
	if versionStr != "1.0.0" {
		t.Fatalf("Latest version mismatch: expected %s, got %s", "1.0.0", versionStr)
	}

	v, versionStr, err = c.LatestVersion("test", "stable") //stable
	if !reflect.DeepEqual(v, testRelease.Versions["1.0.0"]) {
		t.Fatalf("Latest version on stable channel failed: expected %q, got %q", testRelease.Versions["1.0.0"], v)
	}
	if versionStr != "1.0.0" {
		t.Fatalf("Latest version mismatch: expected %s, got %s", "1.0.0", versionStr)
	}

	v, versionStr, err = c.LatestVersion("test", "beta") //stable
	if !reflect.DeepEqual(v, testRelease.Versions["1.0.1-beta"]) {
		t.Fatalf("Latest version on stable channel failed: expected %q, got %q", testRelease.Versions["1.0.1-beta"], v)
	}
	if versionStr != "1.0.1-beta" {
		t.Fatalf("Latest version mismatch: expected %s, got %s", "1.0.1-beta", versionStr)
	}
}
