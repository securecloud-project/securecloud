package checks

import (
	"context"
	"net"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
		valid bool
	}{
		{input: " Example.COM. ", want: "example.com", valid: true},
		{input: "8.8.8.8", want: "8.8.8.8", valid: true},
		{input: "https://example.com", valid: false},
		{input: "example.com:8443", valid: false},
		{input: "localhost", valid: false},
		{input: "127.0.0.1", valid: false},
		{input: "169.254.169.254", valid: false},
		{input: "10.0.0.1", valid: false},
		{input: "::1", valid: false},
		{input: "2001:db8::1", valid: false},
		{input: "64:ff9b::7f00:1", valid: false},
		{input: "64:ff9b:1::a00:1", valid: false},
		{input: "fec0::1", valid: false},
		{input: "2002:7f00:1::", valid: false},
		{input: "not_a_host.example", valid: false},
	} {
		got, err := NormalizeTarget(test.input)
		if test.valid && (err != nil || got != test.want) {
			t.Errorf("NormalizeTarget(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
		if !test.valid && err == nil {
			t.Errorf("NormalizeTarget(%q) unexpectedly succeeded", test.input)
		}
	}
}

type fixedResolver []net.IPAddr

func (r fixedResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) { return r, nil }

func TestSecureDialerRejectsAnyPrivateResolution(t *testing.T) {
	dialer := SecureDialer{Resolver: fixedResolver{
		{IP: net.ParseIP("93.184.216.34")},
		{IP: net.ParseIP("127.0.0.1")},
	}}
	if _, err := dialer.DialContext(context.Background(), "tcp", "example.com:443"); err == nil {
		t.Fatal("DialContext() unexpectedly accepted mixed public/private DNS results")
	}
}
