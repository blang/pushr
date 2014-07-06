package pushr

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"os"
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

func (c *Client) LatestVersion(release string) (*Version, error) {
	//TODO: Implement
	return nil, nil
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
