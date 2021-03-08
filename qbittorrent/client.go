package qbittorrent

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"tghomebot/api"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

//Client qbittorent api
type Client struct {
	client   fasthttp.Client
	url      string
	cookie   []byte
	login    string
	password string
}

//nolint:golint
const (
	QueuedUPState    = "queuedUP"
	CheckingDLState  = "checkingDL"
	DownloadingState = "downloading"
	UploadingState   = "uploading"

	addTorrentsRoute  = "/api/v2/torrents/add"
	torrentsInfoRoute = "/api/v2/torrents/info"
)

//NewAPIClient constructor
func NewAPIClient(url, login, password string) *Client {
	a := &Client{
		login:    login,
		password: password,
		url:      url,
		client:   fasthttp.Client{},
	}
	a.auth()
	go func() {
		for range time.Tick(time.Hour) {
			a.auth()
		}
	}()
	return a
}

//GetTorrentsInfo returning all torrents
func (c *Client) GetTorrentsInfo() ([]api.Torrent, error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(c.url + torrentsInfoRoute)
	req.Header.SetCookie("SID", "e6KimCBra0fsDLFq0pa5B6joVi3XmOFD")
	req.Header.SetBytesV(fasthttp.HeaderCookie, c.cookie)

	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}
	if resp.Header.StatusCode() > 400 {
		c.auth()
		fasthttp.ReleaseResponse(resp)
		resp = fasthttp.AcquireResponse()
		c.client.Do(req, resp)
	}
	result := api.Torrents{}
	err := easyjson.Unmarshal(resp.Body(), &result)
	return result, err
}

//SendMagnet creates torrent downloading by magnet
func (c *Client) SendMagnet(magnet []byte) (err error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	boundary := c.randomBoundary()

	req.SetRequestURI(c.url + addTorrentsRoute)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetMultipartFormBoundary(boundary)
	req.SetBodyRaw(c.getMultipartURLBody([]byte(boundary), magnet))
	req.Header.SetBytesV(fasthttp.HeaderCookie, c.cookie)

	err = fasthttp.Do(req, resp)
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("torrents/add response status code:%d", resp.StatusCode())
	}
	return
}

//SendFile creates torrent downloading by file
func (c *Client) SendFile(file []byte) (err error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	boundary := c.randomBoundary()
	req.SetRequestURI(c.url + addTorrentsRoute)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetMultipartFormBoundary(boundary)
	req.SetBodyRaw(c.getMultipartFileBody([]byte(boundary), file))
	req.Header.SetBytesV(fasthttp.HeaderCookie, c.cookie)

	err = c.client.Do(req, resp)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("torrents/add response status code:%d", resp.StatusCode())
	}
	return err
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

func (c *Client) auth() {
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(c.url + "/api/v2/auth/login")
	req.SetBodyString(fmt.Sprintf("username=%s&password=%s", c.login, c.password))
	req.Header.SetMethod(fasthttp.MethodPost)

	c.client.Do(req, res)
	s := res.Header.Peek(fasthttp.HeaderSetCookie)
	c.cookie = s
}
