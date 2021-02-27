package main

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	openapi "github.com/rannoch/highloadcup2021/client"
	"github.com/valyala/fasthttp"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Client struct {
	httpClient *fasthttp.Client
	baseUrl    string
	Debug      bool
	Slowlog    time.Duration
}

func NewClient(httpClient *fasthttp.Client, baseUrl string) *Client {
	return &Client{httpClient: httpClient, baseUrl: baseUrl}
}

func (client *Client) IssueLicense(coin []int32) (openapi.License, int, error) {
	start := time.Now()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	req.SetRequestURI(client.baseUrl + "/licenses")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	requestBody, err := json.Marshal(coin)
	if err != nil {
		return openapi.License{}, 0, err
	}

	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	err = client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if client.Slowlog != 0 && time.Since(start) > client.Slowlog {
		fmt.Printf("%s took %v\n\n", "IssueLicense", time.Since(start))
	}

	body := resp.Body()

	var license openapi.License

	if err := json.Unmarshal(body, &license); err != nil {
		return openapi.License{}, resp.StatusCode(), err
	}

	return license, resp.StatusCode(), nil
}

func (client *Client) ListLicenses() ([]openapi.License, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/licenses")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	if err := client.httpClient.Do(req, resp); err != nil {
		return nil, 0, err
	}

	body := resp.Body()

	var licenses []openapi.License

	if err := json.Unmarshal(body, &licenses); err != nil {
		return nil, resp.StatusCode(), err
	}

	return licenses, resp.StatusCode(), nil
}

func (client *Client) HealthCheck() (int, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/health-check")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	if err := client.httpClient.Do(req, resp); err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}

func (client *Client) GetBalance() (openapi.Balance, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/balance")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	if err := client.httpClient.Do(req, resp); err != nil {
		return openapi.Balance{}, resp.StatusCode(), err
	}

	var balance openapi.Balance

	if err := json.Unmarshal(resp.Body(), &balance); err != nil {
		return openapi.Balance{}, resp.StatusCode(), err
	}

	return balance, resp.StatusCode(), nil
}

func (client *Client) ExploreArea(area openapi.Area) (openapi.Report, int, error) {
	start := time.Now()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	req.SetRequestURI(client.baseUrl + "/explore")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	requestBody, err := json.Marshal(area)
	if err != nil {
		return openapi.Report{}, 0, err
	}

	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	err = client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if client.Slowlog != 0 && time.Since(start) > client.Slowlog {
		fmt.Printf("%s took %v\n\n", "ExploreArea", time.Since(start))
	}

	if err != nil {
		return openapi.Report{}, resp.StatusCode(), err
	}

	var report openapi.Report

	if err := json.Unmarshal(resp.Body(), &report); err != nil {
		return openapi.Report{}, resp.StatusCode(), err
	}

	return report, resp.StatusCode(), nil
}

func (client *Client) Dig(dig openapi.Dig) ([]string, int, error) {
	start := time.Now()

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/dig")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	requestBody, err := json.Marshal(dig)
	if err != nil {
		return nil, 0, err
	}

	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	err = client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if client.Slowlog != 0 && time.Since(start) > client.Slowlog {
		fmt.Printf("%s took %v\n\n", "Dig", time.Since(start))
	}

	if err != nil {
		return nil, resp.StatusCode(), err
	}

	var treasures []string

	if resp.StatusCode() == 200 {
		if err := json.Unmarshal(resp.Body(), &treasures); err != nil {
			return nil, resp.StatusCode(), err
		}
	}

	return treasures, resp.StatusCode(), err
}

func (client *Client) Cash(treasure string) ([]int32, int, error) {
	start := time.Now()

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/cash")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	req.SetBody([]byte(treasure))

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	_ = client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if client.Slowlog != 0 && time.Since(start) > client.Slowlog {
		fmt.Printf("%s took %v\n\n", "Cash", time.Since(start))
	}

	var coins []int32

	if err := json.Unmarshal(resp.Body(), &coins); err != nil {
		return nil, resp.StatusCode(), err
	}

	return coins, resp.StatusCode(), nil
}
