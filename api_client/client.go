package api_client

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/rannoch/highloadcup2021/miner/model"
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

func (client *Client) IssueLicense(coin []int32) (model.License, int, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	req.SetRequestURI(client.baseUrl + "/licenses")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	requestBody, err := json.Marshal(coin)
	if err != nil {
		return model.License{}, 0, err
	}

	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	err = client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if err != nil {
		return model.License{}, resp.StatusCode(), err
	}

	body := resp.Body()

	var license model.License

	if err := json.Unmarshal(body, &license); err != nil {
		return model.License{}, resp.StatusCode(), err
	}

	return license, resp.StatusCode(), nil
}

func (client *Client) ListLicenses() ([]model.License, int, error) {
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

	var licenses []model.License

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

func (client *Client) GetBalance() (model.Balance, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/balance")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	if err := client.httpClient.Do(req, resp); err != nil {
		return model.Balance{}, resp.StatusCode(), err
	}

	var balance model.Balance

	if err := json.Unmarshal(resp.Body(), &balance); err != nil {
		return model.Balance{}, resp.StatusCode(), err
	}

	return balance, resp.StatusCode(), nil
}

func (client *Client) ExploreArea(area model.Area) (model.Report, int, time.Duration, error) {
	start := time.Now()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	req.SetRequestURI(client.baseUrl + "/explore")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	requestBody, err := area.MarshalJSON()
	if err != nil {
		return model.Report{}, 0, time.Since(start), err
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
		return model.Report{}, resp.StatusCode(), time.Since(start), err
	}

	var report model.Report

	if err := json.Unmarshal(resp.Body(), &report); err != nil {
		return model.Report{}, resp.StatusCode(), time.Since(start), err
	}

	return report, resp.StatusCode(), time.Since(start), nil
}

func (client *Client) Dig(dig model.Dig) ([]string, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/dig")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	defer fasthttp.ReleaseRequest(req) // <- do not forget to release

	requestBody, err := dig.MarshalJSON()
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
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(client.baseUrl + "/cash")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	req.SetBody([]byte(treasure))

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	err := client.httpClient.Do(req, resp)
	if client.Debug {
		fmt.Println(req.String() + "\n" + resp.String())
	}

	if err != nil {
		return nil, resp.StatusCode(), err
	}

	var coins []int32

	if err := json.Unmarshal(resp.Body(), &coins); err != nil {
		return nil, resp.StatusCode(), err
	}

	return coins, resp.StatusCode(), nil
}
