package toolsafety

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSafeHTTPSRetainsHostnameAndPinsSocket(t *testing.T) {
	type requestInfo struct{ host, sni string }
	requests := make(chan requestInfo, 1)
	dials := make(chan string, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- requestInfo{r.Host, r.TLS.ServerName}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := SSRFSafeClient(t.Context(), 5*time.Second)
	transport := client.Transport.(*http.Transport)
	defer transport.CloseIdleConnections()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate()) // httptest's certificate includes example.com.
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	dialer := publicDialer{
		lookup: func(_ context.Context, host string) ([]net.IPAddr, error) {
			if host != "example.com" {
				return nil, fmt.Errorf("unexpected host %q", host)
			}
			return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
		},
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dials <- address
			return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "https://"))
		},
	}
	transport.DialContext = dialer.DialContext
	response, err := client.Get("https://example.com:8443/")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	got, dialAddress := <-requests, <-dials
	if got.host != "example.com:8443" || got.sni != "example.com" || dialAddress != "93.184.216.34:8443" {
		t.Fatalf("host=%q SNI=%q dial=%q", got.host, got.sni, dialAddress)
	}
	// A different hostname must still fail certificate verification.
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "https://"))
	}
	if response, err := client.Get("https://wrong.invalid:8443/"); err == nil {
		_ = response.Body.Close()
		t.Fatal("certificate verification was bypassed")
	}
}

func TestPublicDialRejectsReboundAndMixedAddresses(t *testing.T) {
	for _, addresses := range [][]net.IPAddr{
		{{IP: net.ParseIP("127.0.0.1")}},
		{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("169.254.169.254")}},
		{{IP: net.ParseIP("::ffff:127.0.0.1")}},
	} {
		dialed := false
		dialer := publicDialer{
			lookup: func(context.Context, string) ([]net.IPAddr, error) { return addresses, nil },
			dial: func(context.Context, string, string) (net.Conn, error) {
				dialed = true
				return nil, fmt.Errorf("should not dial")
			},
		}
		if _, err := dialer.DialContext(t.Context(), "tcp", "example.com:443"); err == nil || dialed {
			t.Fatalf("private DNS results reached socket dial: %v", addresses)
		}
	}
}

func TestSafeClientBlocksPrivateRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/metadata", http.StatusFound)
	}))
	defer server.Close()
	client := SSRFSafeClient(t.Context(), 5*time.Second)
	transport := client.Transport.(*http.Transport)
	defer transport.CloseIdleConnections()
	dials := 0
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		dials++
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}
	if response, err := client.Get("http://93.184.216.34/start"); err == nil {
		_ = response.Body.Close()
		t.Fatal("private redirect was followed")
	}
	if dials != 1 {
		t.Fatalf("redirect target was dialed: %d dials", dials)
	}
}

func TestValidateURLPreservesIPv6Authority(t *testing.T) {
	input := "https://[2606:4700:4700::1111]:8443/path"
	got, authority, err := ValidateURLPublic(t.Context(), input)
	if err != nil || got != input || authority != "[2606:4700:4700::1111]:8443" {
		t.Fatalf("URL=%q authority=%q error=%v", got, authority, err)
	}
}
