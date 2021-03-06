package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type HttpClient struct {
	token string
}

type HttpResponse struct {
	Status     string
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

func New() *HttpClient {
	return &HttpClient{}
}

func NewWithToken(token string) *HttpClient {
	return &HttpClient{token: token}
}

func (c *HttpClient) FetchToken(stage, username, password string) error {
	client := &http.Client{}

	in := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}

	out := &struct {
		Token string `json:"token"`
	}{}

	var url string
	if stage == "prod" {
		url = "https://api-auth.users.enlight.skf.com/login"
	} else {
		url = fmt.Sprintf("https://api-auth.%s.users.enlight.skf.com/login", stage)
	}

	bs := new(bytes.Buffer)
	if err := json.NewEncoder(bs).Encode(in); err != nil {
		return errors.Wrapf(err, "Failed marshal body for POST request to endpoint: %s", url)
	}

	req, err := http.NewRequest("POST", url, bs)
	if err != nil {
		return errors.Wrapf(err, "Failed to create POST request to endpoint: %s", url)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "POST request to endpoint: %s failed", url)
	}

	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
		return errors.Wrapf(err, "Failed to unmarshal json response from POST request to endpoint: %s", url)
	}

	c.token = out.Token
	return nil
}

func (c *HttpClient) Get(url string, out interface{}) (*HttpResponse, error) {
	return c.send("GET", url, nil, out)
}

func (c *HttpClient) Post(url string, in interface{}, out interface{}) (*HttpResponse, error) {
	return c.send("POST", url, in, out)
}

func (c *HttpClient) Put(url string, in interface{}, out interface{}) (*HttpResponse, error) {
	return c.send("PUT", url, in, out)
}

func (c *HttpClient) Delete(url string, out interface{}) (*HttpResponse, error) {
	return c.send("DELETE", url, nil, out)
}

func (c *HttpClient) send(method, url string, in interface{}, out interface{}) (*HttpResponse, error) {
	client := &http.Client{}

	bs := new(bytes.Buffer)
	sendBody := in != nil && (method == "POST" || method == "PUT")

	if sendBody {
		if err := json.NewEncoder(bs).Encode(in); err != nil {
			return nil, errors.Wrapf(err, "Failed marshal body for %s request to endpoint: %s", method, url)
		}
	}

	req, err := http.NewRequest(method, url, bs)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create %s request to endpoint: %s", method, url)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", c.token)
	if sendBody {
		req.Header.Set("content-type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "%s request to endpoint: %s failed", method, url)
	}

	r, err := parseHttpResponse(resp)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse response from %s request to endpoint: %s", method, url)
	}

	if out != nil {
		if err = json.Unmarshal(r.Body, out); err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal json response from %s request to endpoint: %s", method, url)
		}
	}

	return r, err
}

func parseHttpResponse(resp *http.Response) (*HttpResponse, error) {
	headers := make(map[string][]string)

	for h, v := range resp.Header {
		headers[h] = v
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &HttpResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}
