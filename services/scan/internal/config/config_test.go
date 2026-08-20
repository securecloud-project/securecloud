package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	config, err := Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	if config.Port != 8080 || config.WorkerCount != 2 || config.QueueDepth != 64 || config.RequestsPerSecond != 20 {
		t.Fatalf("unexpected defaults: %+v", config)
	}
}

func TestLoadRejectsUnsafeConfiguration(t *testing.T) {
	for _, values := range []map[string]string{
		{"PORT": "0"},
		{"SCAN_WORKERS": "100"},
		{"SCAN_NETWORK_TIMEOUT": "0s"},
		{"SCAN_NETWORK_TIMEOUT": "10s", "SCAN_JOB_TIMEOUT": "5s"},
		{"NOTIFICATION_SERVICE_URL": "file:///etc/passwd"},
		{"NOTIFICATION_SERVICE_URL": "https://user:password@example.com"},
		{"AUTH_SERVICE_URL": "file:///etc/passwd"},
		{"SCAN_REQUEST_BURST": "0"},
	} {
		_, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
		if err == nil {
			t.Errorf("Load(%v) unexpectedly succeeded", values)
		}
	}
}
