package pushr

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/blang/semver"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

type Client struct {
	host       string
	readToken  string
	writeToken string
}

func NewClient(host, readToken string, writeToken string) *Client {
	return &Client{
		host:       host,
		readToken:  readToken,
		writeToken: writeToken,
	}
}

type Release struct {
	Versions map[string]*Version `json:"versions"`
}

func NewRelease() *Release {
	return &Release{
		Versions: make(map[string]*Version),
	}
}

type Version struct {
	ContentType string `json:"contenttype"`
	Size        int64  `json:"size"`
	Filename    string `json:"filename"`
}

type ByVersion []semver.Version

func (a ByVersion) Len() int {
	return len(a)
}

func (a ByVersion) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByVersion) Less(i, j int) bool {
	return a[i].LT(a[j])
}

func NewVersion() *Version {
	return &Version{}
}

func (c *Client) Release(release string) (*Release, error) {
	req, err := http.NewRequest("GET", c.cleanHost()+"/releases/"+release, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-PUSHR-TOKEN", c.readToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	binresp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer binresp.Body.Close()
	r := bufio.NewReader(binresp.Body)
	var rel Release
	err = json.NewDecoder(r).Decode(&rel)
	if err != nil {
		return nil, err
	}

	return &rel, nil
}

func (c *Client) LatestVersion(release string, channel string) (*Version, string, error) {
	r, err := c.Release(release)
	if err != nil {
		return nil, "", err
	}
	if channel == "" {
		channel = "stable"
	}

	versions := make([]semver.Version, 0, len(r.Versions))
	for versionStr := range r.Versions {
		v, err := semver.New(versionStr)
		if err == nil {
			versions = append(versions, v)
		}
	}

	sort.Sort(ByVersion(versions))

	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if channel == "stable" {
			if len(v.Pre) == 0 {
				return r.Versions[v.String()], v.String(), nil
			}
		} else {
			// Accept stable release if it's the latest version, otherwise search for specific channel
			if len(v.Pre) == 0 || (len(v.Pre) > 0 && v.Pre[0].String() == channel) {
				return r.Versions[v.String()], v.String(), nil
			}
		}
	}

	return nil, "", errors.New("No version in this channel available")
}

func (c *Client) Version(release string, versionstr string) (*Version, error) {
	req, err := http.NewRequest("GET", c.cleanHost()+"/releases/"+release+"/"+versionstr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-PUSHR-TOKEN", c.readToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	binresp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer binresp.Body.Close()
	r := bufio.NewReader(binresp.Body)
	var version Version
	err = json.NewDecoder(r).Decode(&version)
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (c *Client) Download(release string, versionstr string, filename string) error {
	req, err := http.NewRequest("GET", c.cleanHost()+"/releases/"+release+"/"+versionstr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-PUSHR-TOKEN", c.readToken)
	req.Header.Set("Accept", "application/octet-stream")
	client := &http.Client{}
	binresp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer binresp.Body.Close()
	r := bufio.NewReader(binresp.Body)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// write a chunk
		if _, err := w.Write(buf[:n]); err != nil {
			return err
		}
	}

	if err = w.Flush(); err != nil {
		return err
	}

	return nil
}

func (c *Client) cleanHost() string {
	return strings.TrimSuffix(c.host, "/")
}
