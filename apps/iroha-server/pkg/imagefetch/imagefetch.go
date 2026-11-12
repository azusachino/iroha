// Package imagefetch fetches remote images for the cover route. One Fetcher
// owns the address policy, the content policy, the size limit and the cache.
package imagefetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const (
	// MaxBytes is the largest image body the fetcher accepts.
	MaxBytes = 10 << 20

	fetchTimeout = 10 * time.Second
	maxRedirects = 3
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
	accept       = "image/avif,image/webp,image/apng,image/*;q=0.8"
)

var (
	ErrInvalidURL = errors.New("invalid image url")
	ErrNotImage   = errors.New("upstream body is not an image")
	ErrTooLarge   = errors.New("upstream image exceeds the size limit")
	ErrUpstream   = errors.New("upstream fetch failed")
)

// Image is a fetched image. ContentType comes from sniffing Data, never from
// the upstream header. CacheErr is set when the image was fetched but could not
// be stored; the image is still valid.
type Image struct {
	Data        []byte
	ContentType string
	CacheErr    error
}

// Fetcher fetches images, caching each one on disk keyed by its URL.
type Fetcher struct {
	dir    string
	client *http.Client
	flight singleflight.Group
}

// Option customizes a Fetcher.
type Option func(*Fetcher, *net.Dialer)

// WithDialPolicy replaces the public-address rule as the dial-time check.
// Production code never needs it; tests use it to reach loopback servers.
func WithDialPolicy(allow func(netip.AddrPort) bool) Option {
	return func(f *Fetcher, d *net.Dialer) { d.Control = control(allow) }
}

// New returns a Fetcher caching under dir.
func New(dir string, opts ...Option) *Fetcher {
	f := &Fetcher{dir: dir}
	dialer := &net.Dialer{Timeout: fetchTimeout, Control: control(func(ap netip.AddrPort) bool { return PublicAddress(ap.Addr()) })}
	for _, opt := range opts {
		opt(f, dialer)
	}
	f.client = &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   fetchTimeout,
			ResponseHeaderTimeout: fetchTimeout,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}
	return f
}

// Key identifies a URL in the cache and is suitable as an ETag value.
func Key(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:])
}

// Fetch returns the image at rawURL from the cache or the network. Concurrent
// calls for one URL share a single fetch.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (Image, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Image{}, ErrInvalidURL
	}
	key := Key(rawURL)
	// The shared fetch must not die with the first caller's request.
	detached := context.WithoutCancel(ctx)
	v, err, _ := f.flight.Do(key, func() (any, error) {
		return f.fetch(detached, parsed, key)
	})
	if err != nil {
		return Image{}, err
	}
	return v.(Image), nil
}

func (f *Fetcher) fetch(ctx context.Context, parsed *url.URL, key string) (Image, error) {
	cacheFile := filepath.Join(f.dir, key)
	if data, err := os.ReadFile(cacheFile); err == nil {
		if contentType, ok := sniffImage(data); ok {
			return Image{Data: data, ContentType: contentType}, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Image{}, ErrInvalidURL
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	if strings.Contains(parsed.Host, "weread.qq.com") || strings.Contains(parsed.Host, "myqcloud.com") {
		req.Header.Set("Referer", "https://weread.qq.com/")
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return Image{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
	if resp.ContentLength > MaxBytes {
		return Image{}, ErrTooLarge
	}
	// Read one byte past the limit so an oversized body is rejected, not cut.
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	if len(data) > MaxBytes {
		return Image{}, ErrTooLarge
	}
	contentType, ok := sniffImage(data)
	if !ok {
		return Image{}, ErrNotImage
	}
	return Image{Data: data, ContentType: contentType, CacheErr: f.store(cacheFile, data)}, nil
}

func (f *Fetcher) store(cacheFile string, data []byte) error {
	if err := os.MkdirAll(f.dir, 0o750); err != nil {
		return err
	}
	tmp := cacheFile + ".tmp." + uuid.NewString()
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, cacheFile); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func sniffImage(data []byte) (string, bool) {
	contentType := http.DetectContentType(data)
	return contentType, strings.HasPrefix(contentType, "image/")
}

func control(allow func(netip.AddrPort) bool) func(network, address string, _ syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		addrPort, err := netip.ParseAddrPort(address)
		if err != nil {
			return fmt.Errorf("parse dial address %q: %w", address, err)
		}
		if !allow(addrPort) {
			return fmt.Errorf("dial %s: address is not public", addrPort.Addr())
		}
		return nil
	}
}

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),   // carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // documentation
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // documentation
	netip.MustParsePrefix("203.0.113.0/24"),  // documentation
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved
	netip.MustParsePrefix("2001:db8::/32"),   // documentation
	netip.MustParsePrefix("64:ff9b::/96"),    // NAT64
}

// PublicAddress reports whether ip is a publicly routable unicast address.
// Loopback, private, link-local, unspecified, multicast, broadcast, and other
// reserved ranges are refused.
func PublicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
