package checks

import (
	"net/url"
	"testing"
)

func TestValidHSTS(t *testing.T) {
	if !validHSTS("max-age=31536000; includeSubDomains") {
		t.Fatal("valid HSTS header was rejected")
	}
	for _, value := range []string{"", "includeSubDomains", "max-age=0"} {
		if validHSTS(value) {
			t.Errorf("invalid HSTS header %q was accepted", value)
		}
	}
}

func TestValidHTTPSRedirect(t *testing.T) {
	for _, test := range []struct {
		location string
		valid    bool
	}{
		{location: "https://example.com/path", valid: true},
		{location: "http://example.com/path", valid: false},
		{location: "https://other.example/path", valid: false},
		{location: "https://user@example.com/path", valid: false},
		{location: "https://127.0.0.1/path", valid: false},
	} {
		location, err := url.Parse(test.location)
		if err != nil {
			t.Fatal(err)
		}
		if got := validHTTPSRedirect(location, "example.com"); got != test.valid {
			t.Errorf("validHTTPSRedirect(%q) = %v, want %v", test.location, got, test.valid)
		}
	}
}
