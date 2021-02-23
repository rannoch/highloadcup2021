package main

import (
	"context"
	"fmt"
	"github.com/antihax/optional"
	"github.com/rannoch/highloadcup2021/client"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

func main() {
	configuration := openapi.NewConfiguration()

	getenv := os.Getenv("ADDRESS")
	if getenv == "" {
		getenv = "localhost"
	}

	urlParsed, err := url.Parse("http://" + getenv + ":8000")
	if err != nil {
		log.Fatal(err)
	}
	urlParsed.Scheme = "http"

	configuration.BasePath = urlParsed.String()
	configuration.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	configuration.Debug = true

	apiClient := openapi.NewAPIClient(configuration)
	service := NewMiningService(NewMiner(apiClient))

	err = service.Start(context.Background())
	fmt.Println(err)
}

type Field struct {
	CellsCurrentDepth [3500][3500]int64
}

func NewField() Field {
	return Field{}
}

type Miner struct {
	client  *openapi.APIClient
	license openapi.License
}

func NewMiner(client *openapi.APIClient) Miner {
	return Miner{client: client}
}

type MiningService struct {
	Miner          Miner
	LicenseManager *LicenseManager
	Cash           []int32
}

func NewMiningService(miner Miner) *MiningService {
	m := &MiningService{Miner: miner}
	m.LicenseManager = NewLicenseManager(miner.client)
	return m
}

func (miningService *MiningService) Start(ctx context.Context) error {
	for {
		_, response, _ := miningService.Miner.client.DefaultApi.HealthCheck(ctx)
		if response != nil && response.StatusCode == 200 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	for x := 0; x < 3500; x++ {
		for y := 0; y < 3500; y++ {
			area, response, err := miningService.Miner.client.DefaultApi.ExploreArea(ctx, openapi.Area{
				PosX:  int32(x),
				PosY:  int32(y),
				SizeX: 1,
				SizeY: 1,
			})
			if err != nil || response.StatusCode != 200 || area.Amount == 0 {
				continue
			}

			var depth int32 = 1

			left := area.Amount

			for depth <= 10 && left > 0 {
				// get license
				license := miningService.LicenseManager.GetLicense(ctx)
				// dig
				treasures, digResp, err := miningService.Miner.client.DefaultApi.Dig(ctx, openapi.Dig{
					LicenseID: license.Id,
					PosX:      int32(x),
					PosY:      int32(y),
					Depth:     depth,
				})

				if digResp != nil && digResp.StatusCode == 403 {
					miningService.LicenseManager.DeleteLicense(ctx, license.Id)
					continue
				}

				if digResp != nil && digResp.StatusCode >= 500 {
					continue
				}

				depth++
				license.DigUsed++
				miningService.LicenseManager.UpdateLicense(ctx, license)

				if digResp != nil && digResp.StatusCode == 404 {
					continue
				}

				if err != nil {
					continue
				}

				if len(treasures) > 0 {
					for _, treasure := range treasures {
						cash, _, err := miningService.Miner.client.DefaultApi.Cash(ctx, fmt.Sprintf("\"%s\"", treasure))
						if err != nil {
							continue
						}

						miningService.Cash = append(miningService.Cash, cash...)
					}
				}
			}
		}
	}
	return nil
}

type LicenseId = int32

type LicenseManager struct {
	licenses map[LicenseId]openapi.License
	mutex    sync.RWMutex

	client *openapi.APIClient
}

func NewLicenseManager(client *openapi.APIClient) *LicenseManager {
	l := &LicenseManager{client: client}
	l.licenses = make(map[LicenseId]openapi.License)
	return l
}

func (m *LicenseManager) GetLicense(ctx context.Context) openapi.License {
	if len(m.licenses) == 0 {
		for {
			err := m.issueNewLicense(ctx)
			if err == nil {
				break
			}
		}
	}

	var license openapi.License

	for _, l := range m.licenses {
		license = l
	}

	return license
}

func (m *LicenseManager) UpdateLicense(ctx context.Context, license openapi.License) {
	if license.DigUsed >= license.DigAllowed {
		m.DeleteLicense(ctx, license.Id)
		return
	}

	m.mutex.Lock()
	m.licenses[license.Id] = license
	m.mutex.Unlock()
}

func (m *LicenseManager) DeleteLicense(_ context.Context, id LicenseId) {
	m.mutex.Lock()
	delete(m.licenses, id)
	m.mutex.Unlock()
}

func (m *LicenseManager) issueNewLicense(ctx context.Context) error {
	license, _, err := m.client.DefaultApi.IssueLicense(ctx, &openapi.IssueLicenseOpts{Args: optional.NewInterface([]int32{})})
	if err != nil {
		return err
	}

	m.mutex.Lock()
	m.licenses[license.Id] = license
	m.mutex.Unlock()
	return nil
}
