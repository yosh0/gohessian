package gohessian

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type hessian_request struct {
	body []byte
}

// NewClient return a client of hessian
// host string
// url  string
func NewClient(host, url string) (c Client) {
	host = HostCheck(host)
	return &Client{
		Host: host,
		URL:  url,
	}
}



// Invoke send a request to hessian service and return the result of response
// method string  => hessian service method
// params ...Any  => request param
func (c *Client) Invoke(method string, params ...Any) (interface{}, error) {
	reqURL := c.Host + c.URL
	r := &hessian_request{}
	r.packHead(method)
	for _, v := range params {
		r.packParam(v)
	}
	r.packEnd()

	resp, err := httpPost(reqURL, bytes.NewReader(r.body))
	if err != nil {
		return nil, err
	}

	h := NewHessian(bytes.NewReader(resp))
	v, err := h.Parse()

	if err != nil {
		return nil, err
	}

	return v, nil
}

//httpPost send HTTP POST request, return bytes in body
func httpPost(url string, body io.Reader) (rb []byte, err error) {
	var resp *http.Response
	if resp, err = http.Post(url, "application/binary", body); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(resp.Status)
		return
	}
	defer resp.Body.Close()
	rb, err = ioutil.ReadAll(resp.Body)
	return
}

// packHead pack hessian request head
func (h *hessian_request) packHead(method string) {
	tmp_b, _ := PackUint16(uint16(len(method)))
	h.body = append(h.body, []byte{99, 0, 1, 109}...)
	h.body = append(h.body, tmp_b...)
	h.body = append(h.body, []byte(method)...)
}

// packParam pack param in hessian request
func (h *hessian_request) packParam(p Any) {
	tmp_b, err := Encode(p)
	if err != nil {
		panic(err)
	}
	h.body = append(h.body, tmp_b...)
}

// packEnd pack end of hessian request
func (h *hessian_request) packEnd() {
	h.body = append(h.body, 'z')
}
