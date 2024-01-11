package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	activeUsers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_active_users",
		Help: "Active users",
	})

	userLimit = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_user_limit",
		Help: "License user limit",
	})

	expirationDate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_license_expires_at",
		Help: "Expiration date",
	})

	maximumUserCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_maximum_user_count",
		Help: "Maximum number of users since license started",
	})

	expired = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_license_expired",
		Help: "License expiration status",
	})

	userOverage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_user_overage",
		Help: "Difference between active users and licesed users",
	})

	scrapeSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_scrape_status",
		Help: "Exporter scrape status",
	})
)

type gitlab struct {
	ActiveUsers      float64 `json:"active_users"`
	UserLimit        float64 `json:"user_limit"`
	ExpirationDate   string  `json:"expires_at"`
	MaximumUserCount float64 `json:"maximum_user_count"`
	Expired          bool    `json:"expired"`
	UserOverage      float64 `json:"overage"`
	ExpireSec        float64
	IsExpired        float64
}

func init() {
	prometheus.MustRegister(activeUsers)
	prometheus.MustRegister(userLimit)
	prometheus.MustRegister(expirationDate)
	prometheus.MustRegister(maximumUserCount)
	prometheus.MustRegister(expired)
	prometheus.MustRegister(userOverage)
	prometheus.MustRegister(scrapeSuccess)
}

func validateEnvVars() (string, error) {
	token, isSet := os.LookupEnv("GITLAB_TOKEN")
	if !isSet {
		return "", errors.New("Environment variable GITLAB_TOKEN is not set")
	}

	url, isSet := os.LookupEnv("GITLAB_URL")
	if !isSet {
		return "", errors.New("Environment variable GITLAB_URL is not set")
	}

	return string(url + "/api/v4/license?private_token=" + token), nil
}

func recordMetrics(url string) {
	go func() {
		for {
			body, err := getBody(url)
			if err != nil {
				log.Print(err)
				scrapeSuccess.Set(0)
			} else {
				glab, err := parseGitlab(body)
				if err != nil {
					log.Print(err)
					scrapeSuccess.Set(0)
				} else {
					scrapeSuccess.Set(1)

					activeUsers.Set(glab.ActiveUsers)
					userLimit.Set(glab.UserLimit)
					expirationDate.Set(glab.ExpireSec)

					log.Print("Scrapped successfully")
				}
			}
			time.Sleep(60 * time.Minute)
		}
	}()
}

func getBody(url string) ([]byte, error) {

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			return nil, errors.New("401 unauthorised")
		} else if resp.StatusCode == 403 {
			return nil, errors.New("403 The request requires higher privileges than provided by the access token.")
		} else {
			return nil, errors.New("Unexpected error. HTTP status code: " + strconv.Itoa(resp.StatusCode))
		}
	}

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return []byte(responseData), nil
}

func parseGitlab(textBytes []byte) (gitlab, error) {
	const layoutISO = "2006-01-02"
	g := gitlab{}

	json.Unmarshal(textBytes, &g)

	t, err := time.Parse(layoutISO, g.ExpirationDate)
	if err != nil {
		return g, err
	}
	g.ExpireSec = float64(t.Unix())

	g.IsExpired = isExpired(g.Expired)

	return g, nil
}

func isExpired(expired bool) float64 {
	switch expired {
	case false:
		return 0
	case true:
		return 1
	default:
		return 0
	}
}

func main() {
	url, err := validateEnvVars()
	if err != nil {
		log.Fatal(err)
	}
	recordMetrics(url)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9090", nil))
}
