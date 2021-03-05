package qbittorrent

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"tghomebot/api"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

type Client struct {
	client fasthttp.Client
	url    string
}

const (
	QueuedUPState    = "queuedUP"
	CheckingDLState  = "checkingDL"
	DownloadingState = "downloading"
	UploadingState   = "uploading"

	addTorrentsRoute  = "/api/v2/torrents/add"
	torrentsInfoRoute = "/api/v2/torrents/info"
)

func NewApi(url string) *Client {
	a := &Client{
		url:    url,
		client: fasthttp.Client{},
	}
	return a
}

func (c *Client) GetTorrentsInfo() ([]api.Torrent, error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(c.url + torrentsInfoRoute)
	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}

	result := api.Torrents{}
	err := easyjson.Unmarshal(resp.Body(), &result)
	return result, err
}

func (c *Client) SendMagnet(magnet []byte) (err error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	boundary := c.randomBoundary()

	req.SetRequestURI(c.url + addTorrentsRoute)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetMultipartFormBoundary(boundary)
	req.SetBodyRaw(c.getMultipartUrlBody([]byte(boundary), magnet))

	err = fasthttp.Do(req, resp)
	if resp.StatusCode() != fasthttp.StatusOK {
		return errors.New(fmt.Sprintf("torrents/add response status code:%d", resp.StatusCode()))
	}
	return
}

func (c *Client) SendFile(file []byte) (err error) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	boundary := c.randomBoundary()
	req.SetRequestURI(c.url + addTorrentsRoute)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetMultipartFormBoundary(boundary)
	req.SetBodyRaw(c.getMultipartFileBody([]byte(boundary), file))

	err = c.client.Do(req, resp)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return errors.New(fmt.Sprintf("torrents/add response status code:%d", resp.StatusCode()))
	}
	return err
}

func (c *Client) getMultipartUrlBody(boundary, body []byte) []byte {
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
