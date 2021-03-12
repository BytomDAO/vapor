package clients

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bytom/bytom/errors"
)

func Get(url string, result interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// Post make a post http request to url
func Post(url string, payload []byte, result interface{}) error {
	return PostWithHeader(url, nil, payload, result)
}

// PostWithHeader fill the header using params for a http post request
func PostWithHeader(url string, header map[string]string, payload []byte, result interface{}) error {
	return requestWithHeader("POST", url, header, payload, result)
}

func requestWithHeader(method, url string, header map[string]string, payload []byte, result interface{}) error {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	// set Content-Type in advance, and overwrite Content-Type if provided
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if result == nil {
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}

// baseClient for sending request
type baseClient struct {
	domain string
}

type respTemplate struct {
	Status    string                     `json:"status"`
	Code      string                     `json:"code"`
	Msg       string                     `json:"msg"`
	ErrDetail string                     `json:"error_detail"`
	Data      json.RawMessage            `json:"data,omitempty"`
	Result    map[string]json.RawMessage `json:"result"`
}

func (b *baseClient) request(url string, reqData, respData interface{}) error {
	payload, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	resp := &respTemplate{}
	if reqData == nil {
		err = Get(b.domain+url, resp)
	} else {
		err = Post(b.domain+url, payload, resp)
	}
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		return errors.New(resp.Msg + ". " + resp.ErrDetail)
	}

	data, ok := resp.Result["data"]
	if !ok {
		return errors.New("fail on find resp data")
	}

	return json.Unmarshal(data, respData)
}

type apiClient struct {
	*baseClient
	baseURL string
}

func newApiClient(domain, baseURL string) *apiClient {
	return &apiClient{
		baseClient: &baseClient{domain},
		baseURL:    baseURL,
	}
}
