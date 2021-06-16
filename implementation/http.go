package implementation

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)



type httpResponse struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.0"
	ProtoMajor int    // e.g. 1
	ProtoMinor int    // e.g. 0

	Header http.Header
	Body   string

	ContentLength int64

	TransferEncoding []string

	Close bool

	Uncompressed bool
}

func readBody(responseBody io.ReadCloser) ([]byte, error) {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Debugf("failed to close responseBody reader, Error: %v", err)
		}
	}(responseBody)

	body, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func createResponse(response *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("response has not been provided")
	}
	body, err := readBody(response.Body)
	if err != nil {
		return nil, err
	}

	resp := &httpResponse{
		Status:           response.Status,
		StatusCode:       response.StatusCode,
		Proto:            response.Proto,
		ProtoMajor:       response.ProtoMajor,
		ProtoMinor:       response.ProtoMinor,
		Header:           response.Header,
		Body:             string(body),
		ContentLength:    response.ContentLength,
		TransferEncoding: response.TransferEncoding,
		Close:            response.Close,
		Uncompressed:     response.Uncompressed,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return respBytes, err
}

func sendRequest(method string, urlAsString string, data interface{}) ([]byte, error) {
	postBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	requestBody := bytes.NewBuffer(postBody)

	// Create new http client with predefined options
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	request, err := http.NewRequest(method, urlAsString, requestBody)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	responseBytes, err := createResponse(response, err)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func executeCoreHTTPGetAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	url, ok := request.Parameters[userProviderUrlKey]
	if !ok {
		return nil, errors.New("no url provider for execution")
	}

	response, err := http.Get(url)
	responseBytes, err := createResponse(response, err)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func executeCoreHTTPPostAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	url, ok := request.Parameters[userProviderUrlKey]
	if !ok {
		return nil, errors.New("no url provider for execution")
	}

	contentType, ok := request.Parameters[userProviderContentTypeKey]
	if !ok {
		return nil, errors.New("no content-type provider for execution")
	}

	bodyAsString, ok := request.Parameters[userProviderBodyKey]
	if !ok {
		bodyAsString = ""
	}

	body, err := json.Marshal(bodyAsString)
	if err != nil {
		return nil, err
	}

	response, err := http.Post(url, contentType, bytes.NewBuffer(body))
	responseBytes, err := createResponse(response, err)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func executeCoreHTTPPutAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	url, ok := request.Parameters[userProviderUrlKey]
	if !ok {
		return nil, errors.New("no url provider for execution")
	}

	bodyAsString, ok := request.Parameters[userProviderBodyKey]
	if !ok {
		bodyAsString = ""
	}

	responseBytes, err := sendRequest(http.MethodPut, url, bodyAsString)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func executeCoreHTTPDeleteAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	url, ok := request.Parameters[userProviderUrlKey]
	if !ok {
		return nil, errors.New("no url provider for execution")
	}

	responseBytes, err := sendRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}
