package qbittorrent

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client qBittorrent Api
type Client struct {
	client   http.Client
	url      string
	cookies  []*http.Cookie
	username string
	password string
}

// qBittorrent api routes
const (
	addTorrentsRoute  = "/api/v2/torrents/add"
	torrentsInfoRoute = "/api/v2/torrents/info"
	loginRoute        = "/api/v2/auth/login"
)

//NewAPIClient constructor
func NewAPIClient(url, username, password string) *Client {
	a := &Client{
		username: username,
		password: password,
		url:      url,
		client:   http.Client{},
	}
	a.login()
	go func() {
		for range time.Tick(time.Hour) {
			a.login()
		}
	}()
	return a
}

//GetTorrentsInfo returning all torrents
func (c *Client) GetTorrentsInfo() ([]Torrent, error) {
	req, err := http.NewRequest(http.MethodGet, c.url+torrentsInfoRoute, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	c.addCookies(req)

	resp, err := c.client.Do(req)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		err = c.login()
		if err != nil {
			return nil, fmt.Errorf("c.username: %w", err)
		}
		c.addCookies(req)
		resp, err = c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("client.Do (%s): %w", req.RequestURI, err)
		}
	}
	var result []Torrent
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}
	return result, nil
}

//SendMagnet creates torrent downloading by magnet
func (c *Client) SendMagnet(magnet []byte) (err error) {
	boundary := c.randomBoundary()
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	requestBody := bytes.NewBuffer(c.getMultipartURLBody([]byte(boundary), magnet))
	req, err := http.NewRequest(http.MethodPost, c.url+addTorrentsRoute, requestBody)
	if err != nil {
		return fmt.Errorf("http.NewRequest:%w", err)
	}
	req.Header.Set("content-type", contentType)
	c.addCookies(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("c.clent.Do (%s) :%w", req.RequestURI, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torrents/add response status code:%d", resp.StatusCode)
	}
	return
}

//SendFile creates torrent downloading by file
func (c *Client) SendFile(file []byte) (err error) {
	boundary := c.randomBoundary()
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	requestBody := bytes.NewBuffer(c.getMultipartFileBody([]byte(boundary), file))

	req, err := http.NewRequest(http.MethodPost, c.url+addTorrentsRoute, requestBody)
	if err != nil {
		return fmt.Errorf("http.NewRequest:%w", err)
	}
	req.Header.Set("content-type", contentType)
	c.addCookies(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("clent.Do (%s): %w", req.RequestURI, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("call (%s) returned %d", req.RequestURI, resp.StatusCode)
	}
	return nil
}

func (c *Client) getMultipartURLBody(boundary, body []byte) []byte {
	mimeHeaders := []byte("\r\nContent-Disposition: form-data; name=\"urls\"\r\n\r\n")
	return c.getMultipartBody(boundary, body, mimeHeaders)
}

func (c *Client) getMultipartFileBody(boundary, body []byte) []byte {
	mimeHeaders := []byte("\r\nContent-Disposition: form-data; name=\"torrents\"; filename=\"t.torrent\"\r\nContent-Type: application/octet-stream\r\n\r\n")
	return c.getMultipartBody(boundary, body, mimeHeaders)
}

func (c *Client) getMultipartBody(boundary, body, mimeHeaders []byte) []byte {
	var buf bytes.Buffer
	size := len(body) + len(boundary)*2 + 4*2 + 2 + len(mimeHeaders)
	buf.Grow(size)
	buf.WriteString("--")
	buf.Write(boundary)
	buf.Write(mimeHeaders)
	buf.Write(body)
	buf.WriteString("\r\n--")
	buf.Write(boundary)
	buf.WriteString("--\r\n")
	return buf.Bytes()
}

func (c *Client) randomBoundary() string {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf)
}

func (c *Client) login() error {
	requestBody := bytes.NewBufferString(fmt.Sprintf("username=%s&password=%s", c.username, c.password))
	request, _ := http.NewRequest(http.MethodPost, c.url+loginRoute, requestBody)
	request.Header.Set("Content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := c.client.Do(request)
	if err != nil {
		return err
	}
	c.cookies = resp.Cookies()
	return nil
}

func (c *Client) addCookies(req *http.Request) {
	req.Header.Del("Cookie")
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
}
