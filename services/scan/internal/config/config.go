package config

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Port                   int
	DatabasePath           string
	NotificationServiceURL string
	NetworkTimeout         time.Duration
	ScanTimeout            time.Duration
	NotificationTimeout    time.Duration
	CertificateExpiry      time.Duration
	WorkerCount            int
	QueueDepth             int
}

type lookupFunc func(string) (string, bool)

func Load(lookup lookupFunc) (Config, error) {
	config := Config{
		Port:                   8080,
		DatabasePath:           "./data/scan.db",
		NetworkTimeout:         5 * time.Second,
		ScanTimeout:            20 * time.Second,
		NotificationTimeout:    3 * time.Second,
		CertificateExpiry:      30 * 24 * time.Hour,
		WorkerCount:            2,
		QueueDepth:             64,
		NotificationServiceURL: "",
	}
	var err error
	if value, ok := lookup("PORT"); ok {
		config.Port, err = parseInt("PORT", value, 1, 65535)
	}
	if err == nil {
		if value, ok := lookup("SCAN_DB_PATH"); ok && value != "" {
			config.DatabasePath = value
		}
		if value, ok := lookup("NOTIFICATION_SERVICE_URL"); ok {
			config.NotificationServiceURL = value
			err = validateServiceURL(value)
		}
	}
	for _, item := range []struct {
		name string
		dest *time.Duration
	}{
		{"SCAN_NETWORK_TIMEOUT", &config.NetworkTimeout},
		{"SCAN_JOB_TIMEOUT", &config.ScanTimeout},
		{"NOTIFICATION_TIMEOUT", &config.NotificationTimeout},
		{"CERT_EXPIRY_THRESHOLD", &config.CertificateExpiry},
	} {
		if err != nil {
			break
		}
		if value, ok := lookup(item.name); ok {
			*item.dest, err = parseDuration(item.name, value)
		}
	}
	if err == nil {
		if value, ok := lookup("SCAN_WORKERS"); ok {
			config.WorkerCount, err = parseInt("SCAN_WORKERS", value, 1, 32)
		}
	}
	if err == nil {
		if value, ok := lookup("SCAN_QUEUE_DEPTH"); ok {
			config.QueueDepth, err = parseInt("SCAN_QUEUE_DEPTH", value, 1, 10000)
		}
	}
	if err != nil {
		return Config{}, err
	}
	if config.ScanTimeout < config.NetworkTimeout {
		return Config{}, fmt.Errorf("SCAN_JOB_TIMEOUT must be at least SCAN_NETWORK_TIMEOUT")
	}
	return config, nil
}

func parseInt(name, value string, minimum, maximum int) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minimum, maximum)
	}
	return parsed, nil
}

func parseDuration(name, value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration", name)
	}
	return parsed, nil
}

func validateServiceURL(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("NOTIFICATION_SERVICE_URL must be an HTTP(S) origin without credentials, query, or fragment")
	}
	return nil
}
