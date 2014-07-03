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
	repository string
	token      string
}

const CHANNEL_STABLE = "stable"

func NewClient(host, repository, token string) *Client {
	return &Client{
		host:       host,
		repository: repository,
		token:      token,
	}
}

func (c *Client) LatestRelease(channel string) (*Release, error) {
	if channel == "" {
		channel = CHANNEL_STABLE
	}
	req, err := http.NewRequest("GET", c.cleanHost()+"/repos/"+c.repository+"/releases/"+channel+"/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-PUSHR-TOKEN", c.token)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	binresp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer binresp.Body.Close()
	r := bufio.NewReader(binresp.Body)
	var release Release
	err = json.NewDecoder(r).Decode(&release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) LatestStableRelease() (*Release, error) {
	return c.LatestRelease("stable")
}

func (c *Client) cleanHost() string {
	return strings.TrimSuffix(c.host, "/")
}

func (c *Client) Download(a *Asset, filename string) error {
	req, err := http.NewRequest("GET", c.cleanHost()+"/repos/"+c.repository+"/assets/"+a.ID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-PUSHR-TOKEN", c.token)
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

type Release struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Assets  []*Asset `json:"assets"`
}

type Asset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
}
