package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

const (
	ipLookupApi          = "https://api.ipify.org"
	digitialoceanApi     = "https://api.digitalocean.com/v2/domains"
	envDigitaloceanToken = "DIGITALOCEAN_TOKEN"
	envDomain            = "DOMAIN"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdin, nil))
	targetIp, err := publicIp()

	if err != nil {
		logger.Error("error fetching the public ip", "error", err)
		os.Exit(1)
	}

	logger.Info("fetched public ip", "ip", targetIp)

	client := digitaloceanClient{token: os.Getenv(envDigitaloceanToken)}

	domain := os.Getenv(envDomain)

	records, err := client.getRecords(domain)
	if err != nil {
		logger.Error("error listing domains", "error", err)
	}

	for _, record := range records {
		if record.Type != "A" {
			continue
		}

		if record.Data == targetIp {
			logger.Info("record already has desired ip",
				"domain", domain,
				"record", record.Name,
				"ip", targetIp,
			)
			continue
		}
		logger.Info("record does not have desired ip",
			"domain", domain,
			"record", record.Name,
			"reportetIp", record.Data,
			"targetIp", targetIp,
		)
		record.Data = targetIp
		if err = client.updateRecord(domain, record); err != nil {
			logger.Error("error updating record", "error", err)
		}
	}
}

func publicIp() (string, error) {
	response, err := http.Get(ipLookupApi)
	if err != nil {
		return "", err
	}

	ip, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}

type digitaloceanClient struct {
	token string
}

func (dc digitaloceanClient) getRecords(domain string) ([]record, error) {
	url := fmt.Sprintf("%s/%s/records", digitialoceanApi, domain)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+dc.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Domains []record `json:"domain_records"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result.Domains, nil
}

func (dc digitaloceanClient) updateRecord(domain string, update record) error {
	var data, err = json.Marshal(update)

	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/records/%d", digitialoceanApi, domain, update.Id)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+dc.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("expectet status code 200, got=%d", resp.StatusCode)
	}
	return nil
}

type record struct {
	Id   int
	Type string
	Name string
	Data string
	Ttl  int
}
