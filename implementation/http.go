package implementation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpResponse struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.0"

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
			log.Debugf("failed to close responseBody reader, error: %v", err)
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

func sendRequest(method string, urlAsString string, timeout string, headers map[string]string, cookies map[string]string, data []byte) ([]byte, error) {
	requestBody := bytes.NewBuffer(data)

	timeoutAsNumber, err := strconv.ParseInt(timeout, 10, 64)
	if err != nil {
		return nil, err
	}

	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar, error: %v", err)
	}

	var cookiesList []*http.Cookie
	for name, value := range cookies {
		cookiesList = append(cookiesList, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	parsedUrl, err := url.Parse(urlAsString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request url, error: %v", err)
	}
	cookieJar.SetCookies(parsedUrl, cookiesList)

	// Create new http client with predefined options
	client := &http.Client{
		Jar:     cookieJar,
		Timeout: time.Second * time.Duration(timeoutAsNumber),
	}

	request, err := http.NewRequest(method, urlAsString, requestBody)
	if err != nil {
		return nil, err
	}

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := client.Do(request)
	responseBytes, err := createResponse(response, err)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func getHeaders(contentType string, headers string) map[string]string {
	headerMap := parseStringToMap(headers)
	headerMap["Content-Type"] = contentType

	return headerMap
}

func parseStringToMap(value string) map[string]string {
	stringMap := make(map[string]string)

	split := strings.Split(value, "\n")
	for _, currentParameter := range split {
		if strings.Contains(currentParameter, "=") {
			currentHeaderSplit := strings.Split(currentParameter, "=")
			parameterKey, parameterValue := currentHeaderSplit[0], currentHeaderSplit[1]

			stringMap[parameterKey] = parameterValue
		}
	}
	return stringMap
}

func executeCoreHTTPAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	method, ok := request.Parameters[methodKey]
	if !ok {
		return nil, errors.New("no method provided for execution")
	}

	url, ok := request.Parameters[UrlKey]
	if !ok {
		return nil, errors.New("no url provided for execution")
	}

	timeout, ok := request.Parameters[timeoutKey]
	if !ok {
		timeout = "60"
	}

	contentType, ok := request.Parameters[contentTypeKey]
	if !ok {
		return nil, errors.New("no content-type provided for execution")
	}

	headers, ok := request.Parameters[headersKey]
	if !ok {
		headers = ""
	}

	cookies, ok := request.Parameters[cookiesKey]
	if !ok {
		cookies = ""
	}

	body, ok := request.Parameters[bodyKey]
	if !ok {
		body = ""
	}

	headerMap := getHeaders(contentType, headers)
	cookieMap := parseStringToMap(cookies)

	response, err := sendRequest(method, url, timeout, headerMap, cookieMap, []byte(body))
	if err != nil {
		return nil, err
	}

	return response, nil
}
