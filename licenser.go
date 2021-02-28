package main

import openapi "github.com/rannoch/highloadcup2021/client"

type Licensor struct {
	client *Client

	getLicenseChan   chan<- openapi.License
	licenseIssueChan chan interface{}

	licenses       []openapi.License
	licensesIssued int
}

func NewLicensor(client *Client, getLicenseChan chan<- openapi.License) *Licensor {
	l := &Licensor{client: client, getLicenseChan: getLicenseChan}
	l.licenseIssueChan = make(chan interface{}, 5)

	return l
}

func (licensor *Licensor) GetLicense() openapi.License {
	if len(licensor.licenses) > 0 {
		return licensor.licenses[0]
	}
	return openapi.License{}
}

func (licensor *Licensor) queueLicense() {
	licensor.licensesIssued++
	licensor.licenseIssueChan <- true
}

func (licensor *Licensor) Start() {
	var addLicenseChan = make(chan openapi.License)

	for i := 0; i < 5; i++ {
		go func(licenseIssueChan chan interface{}, addLicenseChan chan openapi.License) {
			for {
				select {
				case <-licenseIssueChan:
					for {
						license, respCode, _ := licensor.client.IssueLicense(PopCoinFromWallet())
						if respCode == 200 {
							addLicenseChan <- license
							break
						}
						if respCode == 409 {
							continue
						}
					}
				}
			}
		}(licensor.licenseIssueChan, addLicenseChan)
	}

	for i := 0; i < 5; i++ {
		licensor.queueLicense()
	}

	for {
		select {
		case licensor.getLicenseChan <- licensor.GetLicense():
			if len(licensor.licenses) > 0 {
				licensor.licenses = licensor.licenses[1:]

				if len(licensor.licenses)+licensor.licensesIssued < 5 {
					licensor.queueLicense()
				}
			}
		case license := <-addLicenseChan:
			licensor.licensesIssued--
			licensor.licenses = append(licensor.licenses, license)
		}
	}
}
