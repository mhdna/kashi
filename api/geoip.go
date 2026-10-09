package api

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// Where a client IP is, for the Security page: country, region, city and
// the network (ISP or hosting company) it belongs to.
//
// It uses DB-IP's free "lite" databases (https://db-ip.com, CC BY 4.0 --
// the page credits them), read from local files, so locating an IP never
// calls out to anyone. The server fetches them itself into GeoIPDir on
// start and then once a day checks for the monthly edition. Until the first
// download lands every lookup comes back empty, which the country filter
// treats as "unknown, let through".

const (
	defaultGeoIPDir   = "./storage/geoip"
	geoIPCityFile     = "dbip-city-lite.mmdb"
	geoIPASNFile      = "dbip-asn-lite.mmdb"
	geoIPDownloadURL  = "https://download.db-ip.com/free/dbip-%s-lite-%s.mmdb.gz"
	geoIPRefreshAfter = 32 * 24 * time.Hour
)

type geoLocation struct {
	CountryCode string
	Country     string
	Region      string
	City        string
	ASN         uint
	Network     string
}

type geoIP struct {
	dir  string
	mu   sync.RWMutex
	city *maxminddb.Reader
	asn  *maxminddb.Reader
}

func newGeoIP(dir string) *geoIP {
	if dir == "" {
		dir = defaultGeoIPDir
	}
	g := &geoIP{dir: dir}
	g.open()
	return g
}

// open (re)loads whichever database files are on disk.
func (g *geoIP) open() {
	city, _ := maxminddb.Open(filepath.Join(g.dir, geoIPCityFile))
	asn, _ := maxminddb.Open(filepath.Join(g.dir, geoIPASNFile))
	g.mu.Lock()
	old := []*maxminddb.Reader{g.city, g.asn}
	g.city, g.asn = city, asn
	g.mu.Unlock()
	for _, r := range old {
		if r != nil {
			r.Close()
		}
	}
}

func (g *geoIP) lookup(ipText string) geoLocation {
	var loc geoLocation
	ip := net.ParseIP(ipText)
	if ip == nil || g == nil {
		return loc
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.city != nil {
		var rec struct {
			Country struct {
				ISOCode string            `maxminddb:"iso_code"`
				Names   map[string]string `maxminddb:"names"`
			} `maxminddb:"country"`
			Subdivisions []struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"subdivisions"`
			City struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"city"`
		}
		if err := g.city.Lookup(ip, &rec); err == nil {
			loc.CountryCode = rec.Country.ISOCode
			loc.Country = rec.Country.Names["en"]
			if len(rec.Subdivisions) > 0 {
				loc.Region = rec.Subdivisions[0].Names["en"]
			}
			loc.City = rec.City.Names["en"]
		}
	}
	if g.asn != nil {
		var rec struct {
			Number uint   `maxminddb:"autonomous_system_number"`
			Org    string `maxminddb:"autonomous_system_organization"`
		}
		if err := g.asn.Lookup(ip, &rec); err == nil {
			loc.ASN, loc.Network = rec.Number, rec.Org
		}
	}
	return loc
}

// runUpdates downloads the databases when they're missing or older than a
// month, checking once a day until ctx ends.
func (g *geoIP) runUpdates(ctx context.Context) {
	for {
		changed := false
		for kind, file := range map[string]string{"city": geoIPCityFile, "asn": geoIPASNFile} {
			path := filepath.Join(g.dir, file)
			if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < geoIPRefreshAfter {
				continue
			}
			if err := downloadGeoIP(ctx, kind, path); err != nil {
				log.Printf("geoip: %s database: %v", kind, err)
				continue
			}
			log.Printf("geoip: %s database updated", kind)
			changed = true
		}
		if changed {
			g.open()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(24 * time.Hour):
		}
	}
}

// downloadGeoIP fetches this month's edition (or last month's, early in a
// month before the new one is published) and swaps it in atomically.
func downloadGeoIP(ctx context.Context, kind, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	now := time.Now().UTC()
	var lastErr error
	for _, month := range []time.Time{now, now.AddDate(0, -1, 0)} {
		url := fmt.Sprintf(geoIPDownloadURL, kind, month.Format("2006-01"))
		if err := downloadGunzip(ctx, url, dest); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func downloadGunzip(ctx context.Context, url, dest string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".geoip-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, gz); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Refuse a file the reader can't open rather than replacing a good one.
	if r, err := maxminddb.Open(tmp.Name()); err != nil {
		return fmt.Errorf("%s: not a database: %w", url, err)
	} else {
		r.Close()
	}
	return os.Rename(tmp.Name(), dest)
}
