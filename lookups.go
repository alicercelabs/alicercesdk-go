package alicercelabs

import (
	"context"
	"net/url"
	"strconv"
)

// IPService is the client's IP geolocation API — client.IP.
type IPService struct{ c *Client }

// IPResult is one IP lookup's answer.
type IPResult struct {
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	CountryCd string  `json:"country_code"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Timezone  string  `json:"timezone"`
	ISP       string  `json:"isp"`
	Org       string  `json:"org"`
	ASN       string  `json:"asn"`
}

// Lookup geolocates a specific public IPv4/IPv6 address.
func (s *IPService) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	var out IPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ip/"+ip, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Self geolocates the caller — the IP the request itself came from.
func (s *IPService) Self(ctx context.Context) (*IPResult, error) {
	var out IPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ip/self", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CEPService is the client's CEP (Brazilian postal code) API — client.CEP.
type CEPService struct{ c *Client }

// CEPResult is one address lookup's answer.
type CEPResult struct {
	CEP        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Cidade     string `json:"cidade"`
	UF         string `json:"uf"`
	DDD        string `json:"ddd,omitempty"`
}

// CEPGetOptions are Get's optional query parameters.
type CEPGetOptions struct {
	// DDD enriches the response with the city's phone area code.
	DDD bool
	// Rota adds route/distance fields — see the CEP docs page.
	Rota bool
}

// Get looks up an address by CEP. opts may be nil.
func (s *CEPService) Get(ctx context.Context, cep string, opts *CEPGetOptions) (*CEPResult, error) {
	q := url.Values{}
	if opts != nil {
		if opts.DDD {
			q.Set("ddd", "true")
		}
		if opts.Rota {
			q.Set("rota", "true")
		}
	}
	var out CEPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cep/"+cep, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Search is a reverse lookup: street name -> CEPs, when you don't have the
// code yet.
func (s *CEPService) Search(ctx context.Context, uf, cidade, logradouro string) ([]CEPResult, error) {
	q := url.Values{"uf": {uf}, "cidade": {cidade}, "logradouro": {logradouro}}
	var out []CEPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cep/busca", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Cities lists every city AlicerceLabs has CEP data for in a given state.
func (s *CEPService) Cities(ctx context.Context, uf string) ([]string, error) {
	q := url.Values{"uf": {uf}}
	var out []string
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cep/cidades", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Neighborhoods lists every neighborhood in a given city.
func (s *CEPService) Neighborhoods(ctx context.Context, uf, cidade string) ([]string, error) {
	q := url.Values{"uf": {uf}, "cidade": {cidade}}
	var out []string
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cep/bairros", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CEPDistance is Distance's answer.
type CEPDistance struct {
	DistanceKM  float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min,omitempty"`
}

// Distance returns the straight-line distance between two CEPs — or, with
// rota=true, driving distance/duration too.
func (s *CEPService) Distance(ctx context.Context, origem, destino string, rota bool) (*CEPDistance, error) {
	q := url.Values{}
	if rota {
		q.Set("rota", "true")
	}
	var out CEPDistance
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cep/distance/"+origem+"/"+destino, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Bulk looks up several CEPs in one call. Costs one rate-limit unit per
// CEP requested, not one per call — see the CEP docs page.
func (s *CEPService) Bulk(ctx context.Context, ceps []string) (map[string]any, error) {
	body, err := jsonBody(map[string]any{"ceps": ceps})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/cep/lote", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DNSService is the client's DNS API — client.DNS.
type DNSService struct{ c *Client }

// DNSResult is one domain's DNS lookup answer.
type DNSResult struct {
	Domain        string `json:"domain"`
	Authoritative struct {
		Servers []string `json:"servers"`
	} `json:"authoritative"`
	Security struct {
		BlockedBig  bool `json:"blocked_big"`
		BlockedNSFW bool `json:"blocked_nsfw"`
	} `json:"security"`
}

// Lookup returns A, AAAA, NS, MX, TXT and CNAME records for a domain, plus
// an ads/NSFW blocklist check.
func (s *DNSService) Lookup(ctx context.Context, domain string) (*DNSResult, error) {
	var out DNSResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/dns/"+domain, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EmailService is the client's email verification API — client.Email.
type EmailService struct{ c *Client }

// EmailResult is one email verification's answer.
type EmailResult struct {
	Email      string `json:"email"`
	Valid      bool   `json:"valid"`
	MXFound    bool   `json:"mx_found"`
	Disposable bool   `json:"disposable"`
}

// Verify checks an email's syntax, MX records and (if enabled
// server-side) does an SMTP probe.
func (s *EmailService) Verify(ctx context.Context, email string) (*EmailResult, error) {
	q := url.Values{"email": {email}}
	var out EmailResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/email/verify", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SSLService is the client's TLS certificate check API — client.SSL.
type SSLService struct{ c *Client }

// SSLResult is one certificate check's answer.
type SSLResult struct {
	Domain          string   `json:"domain"`
	Issuer          string   `json:"issuer"`
	Subject         string   `json:"subject"`
	NotBefore       string   `json:"not_before"`
	NotAfter        string   `json:"not_after"`
	DaysUntilExpiry int      `json:"days_until_expiry"`
	SANs            []string `json:"sans"`
	IsExpired       bool     `json:"is_expired"`
	IsSelfSigned    bool     `json:"is_self_signed"`
	IsValid         bool     `json:"is_valid"`
}

// Check returns validity, issuer and SANs for a domain's TLS certificate.
func (s *SSLService) Check(ctx context.Context, domain string) (*SSLResult, error) {
	var out SSLResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ssl/"+domain, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MapsService is the client's geocoding/routing API — client.Maps.
type MapsService struct{ c *Client }

// GeocodeResult is Geocode/Reverse's answer.
type GeocodeResult struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// RouteResult is Route's answer.
type RouteResult struct {
	DistanceKM  float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
}

// Geocode turns an address into coordinates.
func (s *MapsService) Geocode(ctx context.Context, address string) (*GeocodeResult, error) {
	q := url.Values{"address": {address}}
	var out GeocodeResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/maps/geocode", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Reverse turns coordinates into an address.
func (s *MapsService) Reverse(ctx context.Context, lat, lon float64) (*GeocodeResult, error) {
	q := url.Values{"lat": {strconv.FormatFloat(lat, 'f', -1, 64)}, "lon": {strconv.FormatFloat(lon, 'f', -1, 64)}}
	var out GeocodeResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/maps/reverse", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Route returns driving distance/duration between two "lat,lon" points.
func (s *MapsService) Route(ctx context.Context, from, to string) (*RouteResult, error) {
	q := url.Values{"from": {from}, "to": {to}}
	var out RouteResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/maps/route", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TrustService is the client's domain trust score API — client.Trust.
type TrustService struct{ c *Client }

// TrustResult is Check's answer — see https://alicercelabs.com.br/apis/trust
// for what each field in Signals means.
type TrustResult struct {
	Domain         string         `json:"domain"`
	Score          int            `json:"score"`
	Verdict        string         `json:"verdict"`
	PointsEarned   int            `json:"points_earned"`
	PointsPossible int            `json:"points_possible"`
	Signals        map[string]any `json:"signals"`
}

// Check returns a composite 0-100 trust score for a domain — SSL, DNS
// blocklist, malware history, domain age, and (for .br domains, or when
// you pass cnpj) business registration status. Pass an empty cnpj to omit
// it.
func (s *TrustService) Check(ctx context.Context, domain, cnpj string) (*TrustResult, error) {
	q := url.Values{}
	if cnpj != "" {
		q.Set("cnpj", cnpj)
	}
	var out TrustResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/trust/"+domain, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
