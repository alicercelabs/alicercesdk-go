package alicercelabs

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// IPService is the client's IP Intelligence API — client.IP.
// Capability-based enrichment: geo, ASN, cloud and privacy signals are
// each resolved from their own best available source, independently — a
// field with no evidence is nil, never a fabricated zero value.
type IPService struct{ c *Client }

// IPResult is one IP lookup's answer.
type IPResult struct {
	IP       string  `json:"ip"`
	Version  int     `json:"version"` // 4 or 6
	Scope    string  `json:"scope"`   // "public", "private", "loopback", "carrier_grade_nat", ...
	Routable bool    `json:"routable"`
	Hostname *string `json:"hostname"` // reverse DNS — nil unless ?include=hostname was used

	Location *IPLocation `json:"location"` // nil for non-routable scopes, or if no geo source had data
	Network  *IPNetwork  `json:"network"`
	Privacy  IPPrivacy   `json:"privacy"`
	Traits   IPTraits    `json:"traits"`

	Meta IPMeta `json:"meta"`
}

type IPContinent struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type IPCountry struct {
	Code string `json:"code"` // ISO 3166-1 alpha-2
	Name string `json:"name"`
	IsEU bool   `json:"is_eu"`
}

type IPRegion struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type IPLocation struct {
	Continent        *IPContinent `json:"continent"`
	Country          *IPCountry   `json:"country"`
	Region           *IPRegion    `json:"region"`
	City             *string      `json:"city"`
	PostalCode       *string      `json:"postal_code"`
	Latitude         *float64     `json:"latitude"`
	Longitude        *float64     `json:"longitude"`
	AccuracyRadiusKM *int         `json:"accuracy_radius_km"`
	GeonameID        *int         `json:"geoname_id"`
	Timezone         *string      `json:"timezone"`
}

type IPCloud struct {
	Provider string  `json:"provider"` // "aws", "gcp", "azure", "cloudflare", ...
	Service  *string `json:"service"`
	Region   *string `json:"region"`
}

type IPRPKI struct {
	Status    string  `json:"status"` // "valid", "invalid", "not_found", "unknown"
	OriginASN *int    `json:"origin_asn"`
	Prefix    *string `json:"prefix"`
}

type IPNetwork struct {
	CIDR         *string  `json:"cidr"` // the matched prefix, when a longest-prefix-match source contributed
	ASN          *int     `json:"asn"`  // a real number, never a composed "AS<n>" string
	ASNName      *string  `json:"asn_name"`
	ASNDomain    *string  `json:"asn_domain"`
	Organization *string  `json:"organization"`
	ISP          *string  `json:"isp"`
	RIR          *string  `json:"rir"`
	Type         string   `json:"type"` // "isp", "hosting", "business", "cdn", "mobile", ...
	Cloud        *IPCloud `json:"cloud"`
	RPKI         *IPRPKI  `json:"rpki"`
}

type IPVPN struct {
	Detected   bool    `json:"detected"`
	Provider   *string `json:"provider"`
	Confidence string  `json:"confidence"` // "high", "medium", "low", "unknown"
	LastSeen   *string `json:"last_seen"`
}

type IPProxy struct {
	Detected   bool    `json:"detected"`
	Type       *string `json:"type"`
	Confidence string  `json:"confidence"`
	LastSeen   *string `json:"last_seen"`
}

type IPTor struct {
	Detected bool  `json:"detected"`
	ExitNode *bool `json:"exit_node"`
}

type IPRelay struct {
	Detected bool    `json:"detected"`
	Provider *string `json:"provider"`
}

type IPResidentialProxy struct {
	Detected   bool    `json:"detected"`
	Provider   *string `json:"provider"`
	Confidence string  `json:"confidence"`
}

// IPPrivacy holds anonymization signals — Detected is always known (never
// a pointer): "checked, not detected" is itself informative.
type IPPrivacy struct {
	Anonymous        *bool              `json:"anonymous"`
	VPN              IPVPN              `json:"vpn"`
	Proxy            IPProxy            `json:"proxy"`
	Tor              IPTor              `json:"tor"`
	Relay            IPRelay            `json:"relay"`
	ResidentialProxy IPResidentialProxy `json:"residential_proxy"`
}

type IPTraits struct {
	Hosting    *bool `json:"hosting"`
	Datacenter *bool `json:"datacenter"`
	Mobile     *bool `json:"mobile"`
	Satellite  *bool `json:"satellite"`
	Crawler    *bool `json:"crawler"`
	Bogon      bool  `json:"bogon"` // true for non-public scopes — always known, never nil
}

type IPConfidence struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
	Network string `json:"network"`
	Privacy string `json:"privacy"`
}

type IPMeta struct {
	UpdatedAt  string              `json:"updated_at"`
	Confidence IPConfidence        `json:"confidence"`
	Sources    map[string][]string `json:"sources,omitempty"` // only with IncludeSourceDetails
}

// IPLookupOptions are optional query parameters shared by Lookup, Self and
// Batch. A nil *IPLookupOptions is equivalent to the zero value.
type IPLookupOptions struct {
	// Fields, when non-empty, restricts the response to these dot-notation
	// paths (e.g. "location.country", "network.asn") — "ip" is always kept.
	Fields []string
	// IncludeSourceDetails adds Meta.Sources (which provider/dataset
	// answered each category) to the response.
	IncludeSourceDetails bool
	// Lang selects the language for geographic names only (continent,
	// country, region, city) — e.g. "pt-BR". Empty uses the API default (en).
	Lang string
}

func (o *IPLookupOptions) query() url.Values {
	if o == nil {
		return nil
	}
	q := url.Values{}
	if len(o.Fields) > 0 {
		q.Set("fields", strings.Join(o.Fields, ","))
	}
	if o.IncludeSourceDetails {
		q.Set("include", "source_details")
	}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	return q
}

// Lookup resolves a specific public IPv4/IPv6 address. A private/reserved
// address is not an error — it comes back as a partial profile (Scope/
// Routable set, the rest nil). opts may be nil.
func (s *IPService) Lookup(ctx context.Context, ip string, opts *IPLookupOptions) (*IPResult, error) {
	var out IPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ip/"+ip, opts.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Self resolves the caller — the IP the request itself came from. opts
// may be nil.
func (s *IPService) Self(ctx context.Context, opts *IPLookupOptions) (*IPResult, error) {
	var out IPResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ip/self", opts.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IPBatchItem is one entry of Batch's answer — Data is set on success,
// Error otherwise (never both).
type IPBatchItem struct {
	IP      string        `json:"ip"`
	Success bool          `json:"success"`
	Data    *IPResult     `json:"data,omitempty"`
	Error   *IPBatchError `json:"error,omitempty"`
}

type IPBatchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Batch resolves up to 100 IPs in one call. Each address is resolved
// independently — one malformed entry never fails the whole batch, it
// just gets Success:false in its own slot. opts may be nil.
func (s *IPService) Batch(ctx context.Context, ips []string, opts *IPLookupOptions) ([]IPBatchItem, error) {
	body, err := jsonBody(map[string]any{"ips": ips})
	if err != nil {
		return nil, err
	}
	var out struct {
		Results []IPBatchItem `json:"results"`
	}
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/ip/batch", opts.query(), body, &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// CEPService is the client's CEP (Brazilian postal code) API — client.CEP.
type CEPService struct{ c *Client }

// CEPResult is one address lookup's answer. Municipio (not "cidade") is
// the field name the API itself uses.
type CEPResult struct {
	CEP              string `json:"cep"`
	Logradouro       string `json:"logradouro"`
	Complemento      string `json:"complemento,omitempty"`
	Bairro           string `json:"bairro"`
	Municipio        string `json:"municipio"`
	MunicipioCodIBGE int64  `json:"municipio_cod_ibge,omitempty"`
	UF               string `json:"uf"`
	Nome             string `json:"nome,omitempty"`
	DDD              string `json:"ddd,omitempty"`
}

// CEPGetOptions are Get's optional query parameters.
type CEPGetOptions struct {
	// DDD enriches the response with the city's phone area code.
	DDD bool
}

// Get looks up an address by CEP. opts may be nil.
func (s *CEPService) Get(ctx context.Context, cep string, opts *CEPGetOptions) (*CEPResult, error) {
	q := url.Values{}
	if opts != nil && opts.DDD {
		q.Set("ddd", "true")
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

// CNPJService is the client's CNPJ (Brazilian company registry) API —
// client.CNPJ. Field names match the API's own (Portuguese, same layout
// the Federal Revenue itself uses), not a translated set.
type CNPJService struct{ c *Client }

type CNPJCNAE struct {
	Codigo    int    `json:"codigo"`
	Descricao string `json:"descricao"`
}

type CNPJEndereco struct {
	Logradouro      string `json:"logradouro,omitempty"`
	Numero          string `json:"numero,omitempty"`
	Complemento     string `json:"complemento,omitempty"`
	Bairro          string `json:"bairro,omitempty"`
	CEP             string `json:"cep,omitempty"`
	Municipio       string `json:"municipio,omitempty"`
	CodigoMunicipio int    `json:"codigo_municipio_ibge,omitempty"`
	UF              string `json:"uf,omitempty"`
}

// CNPJSocio is one entry in the QSA (quadro de sócios e administradores)
// — CPFCNPJMascarado already comes masked from the API (***XXXXXX**),
// never plaintext.
type CNPJSocio struct {
	Nome             string `json:"nome"`
	Qualificacao     string `json:"qualificacao"`
	DataEntrada      string `json:"data_entrada,omitempty"`
	CPFCNPJMascarado string `json:"cpf_cnpj_mascarado,omitempty"`
	FaixaEtaria      string `json:"faixa_etaria,omitempty"`
}

// CNPJResult is one company lookup's answer. Meta.Fonte says which
// source answered ("local" or "brasilapi") — see the CNPJ API's docs for
// why there are two.
type CNPJResult struct {
	CNPJ   string `json:"cnpj"`
	Matriz bool   `json:"matriz"`

	RazaoSocial  string `json:"razao_social"`
	NomeFantasia string `json:"nome_fantasia,omitempty"`

	SituacaoCadastral          int    `json:"situacao_cadastral"`
	DescricaoSituacaoCadastral string `json:"descricao_situacao_cadastral"`
	DataSituacaoCadastral      string `json:"data_situacao_cadastral,omitempty"`
	MotivoSituacaoCadastral    int    `json:"motivo_situacao_cadastral,omitempty"`
	DescricaoMotivoSituacao    string `json:"descricao_motivo_situacao_cadastral,omitempty"`

	DataInicioAtividade string `json:"data_inicio_atividade,omitempty"`

	NaturezaJuridica       string `json:"natureza_juridica,omitempty"`
	CodigoNaturezaJuridica int    `json:"codigo_natureza_juridica,omitempty"`

	Porte         string  `json:"porte,omitempty"`
	CapitalSocial float64 `json:"capital_social"`

	CNAEFiscal      CNPJCNAE   `json:"cnae_fiscal"`
	CNAESecundarios []CNPJCNAE `json:"cnaes_secundarios,omitempty"`

	Endereco CNPJEndereco `json:"endereco"`

	Telefone string `json:"telefone,omitempty"`
	Email    string `json:"email,omitempty"`

	OpcaoPeloSimples *bool `json:"opcao_pelo_simples"`
	OpcaoPeloMEI     *bool `json:"opcao_pelo_mei"`

	QSA []CNPJSocio `json:"qsa,omitempty"`

	Meta struct {
		Fonte string `json:"fonte"`
	} `json:"meta"`
}

// Get looks up a company by CNPJ, with or without punctuation
// ("33683111000280" or "33.683.111/0002-80").
func (s *CNPJService) Get(ctx context.Context, cnpj string) (*CNPJResult, error) {
	var out CNPJResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cnpj/"+cnpj, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CPFService is the client's CPF API — client.CPF. Check-digit
// validation and issuing fiscal region only, no external source
// involved (see the API's own docs).
type CPFService struct{ c *Client }

// CPFResult is Get's answer.
type CPFResult struct {
	CPF          string   `json:"cpf"`
	Valido       bool     `json:"valido"`
	RegiaoFiscal int      `json:"regiao_fiscal"`
	Estados      []string `json:"estados"`
}

// Get validates a CPF (with or without punctuation) and resolves its
// issuing fiscal region.
func (s *CPFService) Get(ctx context.Context, cpf string) (*CPFResult, error) {
	var out CPFResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cpf/"+cpf, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FeriadosService is the client's national holidays API —
// client.Feriados.
type FeriadosService struct{ c *Client }

// Holiday is one entry of List's answer.
type Holiday struct {
	Data string `json:"data"` // YYYY-MM-DD
	Nome string `json:"nome"`
	Tipo string `json:"tipo"` // "fixo" or "movel"
}

// List returns Brazil's national holidays (fixed and moveable) for a
// given year, 1900 through 2199.
func (s *FeriadosService) List(ctx context.Context, ano int) ([]Holiday, error) {
	var out []Holiday
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/feriados/"+strconv.Itoa(ano), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DiasUteisService is the client's business-day API — client.DiasUteis.
// Derives from Feriados internally, server-side.
type DiasUteisService struct{ c *Client }

// DiasUteisResult is Count's answer.
type DiasUteisResult struct {
	DiasUteis []string `json:"dias_uteis"`
	Total     int      `json:"total"`
}

// DiasUteisOptions configures Count. The zero value counts national
// holidays as non-business days (the API's own default).
type DiasUteisOptions struct {
	// ExcluirFeriados, when true, counts only weekends as non-business
	// days — national holidays count as business days.
	ExcluirFeriados bool
}

// Count returns every business day between dataInicial and dataFinal
// (both YYYY-MM-DD, inclusive), skipping weekends and, unless opts says
// otherwise, national holidays. Range capped at 10 years by the API.
func (s *DiasUteisService) Count(ctx context.Context, dataInicial, dataFinal string, opts *DiasUteisOptions) (*DiasUteisResult, error) {
	q := url.Values{"data_inicial": {dataInicial}, "data_final": {dataFinal}}
	if opts != nil && opts.ExcluirFeriados {
		q.Set("feriados", "false")
	}
	var out DiasUteisResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/diasuteis", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ISBNService is the client's ISBN API — client.ISBN.
type ISBNService struct{ c *Client }

// ISBNResult is Get's answer. Meta.Fonte says which source answered
// ("open-library" or "brasilapi").
type ISBNResult struct {
	ISBN      string   `json:"isbn"`
	Titulo    string   `json:"titulo"`
	Subtitulo string   `json:"subtitulo,omitempty"`
	Autores   []string `json:"autores,omitempty"`
	Editora   string   `json:"editora,omitempty"`
	AnoPub    string   `json:"ano_publicacao,omitempty"`
	Paginas   int      `json:"paginas,omitempty"`
	CapaURL   string   `json:"capa_url,omitempty"`
	Meta      struct {
		Fonte string `json:"fonte"`
	} `json:"meta"`
}

// Get looks up a book's metadata by ISBN-10 or ISBN-13, with or without
// hyphens.
func (s *ISBNService) Get(ctx context.Context, isbn string) (*ISBNResult, error) {
	var out ISBNResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/isbn/"+isbn, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IBGEService is the client's IBGE API — client.IBGE. Regions, states,
// municipalities and CNAE (business activity) classes, no fallback (the
// source is already the official one).
type IBGEService struct{ c *Client }

type IBGERegiao struct {
	ID    int    `json:"id"`
	Sigla string `json:"sigla"`
	Nome  string `json:"nome"`
}

type IBGEEstado struct {
	ID     int        `json:"id"`
	Sigla  string     `json:"sigla"`
	Nome   string     `json:"nome"`
	Regiao IBGERegiao `json:"regiao"`
}

type IBGEMunicipio struct {
	CodigoIBGE int    `json:"codigo_ibge"`
	Nome       string `json:"nome"`
	UF         string `json:"uf"`
}

type IBGECNAECategoria struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
}

type IBGECNAEClasse struct {
	Codigo    string            `json:"codigo"`
	Descricao string            `json:"descricao"`
	Grupo     IBGECNAECategoria `json:"grupo"`
	Divisao   IBGECNAECategoria `json:"divisao"`
	Secao     IBGECNAECategoria `json:"secao"`
}

// Regioes lists Brazil's 5 macro-regions.
func (s *IBGEService) Regioes(ctx context.Context) ([]IBGERegiao, error) {
	var out []IBGERegiao
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/regioes", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Estados lists every Brazilian state.
func (s *IBGEService) Estados(ctx context.Context) ([]IBGEEstado, error) {
	var out []IBGEEstado
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/uf", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Estado looks up one state by sigla ("SP") or IBGE code ("35").
func (s *IBGEService) Estado(ctx context.Context, codigo string) (*IBGEEstado, error) {
	var out IBGEEstado
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/uf/"+codigo, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Municipios lists every municipality of a state (sigla, e.g. "SP").
func (s *IBGEService) Municipios(ctx context.Context, uf string) ([]IBGEMunicipio, error) {
	var out []IBGEMunicipio
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/municipios/"+uf, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CNAEClasses lists every CNAE (business activity) class.
func (s *IBGEService) CNAEClasses(ctx context.Context) ([]IBGECNAEClasse, error) {
	var out []IBGECNAEClasse
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/cnae", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CNAE looks up one CNAE class by code (e.g. "01113").
func (s *IBGEService) CNAE(ctx context.Context, codigo string) (*IBGECNAEClasse, error) {
	var out IBGECNAEClasse
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/ibge/cnae/"+codigo, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CEPBulkResult is one entry of Bulk's answer — Endereco is set on
// success, Erro is set otherwise (never both).
type CEPBulkResult struct {
	CEP      string     `json:"cep"`
	Endereco *CEPResult `json:"endereco,omitempty"`
	Erro     string     `json:"erro,omitempty"`
}

// Bulk looks up several CEPs in one call. Costs one rate-limit unit per
// CEP requested, not one per call — see the CEP docs page.
func (s *CEPService) Bulk(ctx context.Context, ceps []string) ([]CEPBulkResult, error) {
	body, err := jsonBody(map[string]any{"ceps": ceps})
	if err != nil {
		return nil, err
	}
	var out []CEPBulkResult
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
