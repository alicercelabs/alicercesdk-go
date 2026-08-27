# alicercesdk-go

[![CI](https://github.com/alicercelabs/alicercesdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/alicercelabs/alicercesdk-go/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/alicercelabs/alicercesdk-go/main/.github/badges/coverage.json)](https://github.com/alicercelabs/alicercesdk-go/actions/workflows/ci.yml)

SDK oficial em Go para a [AlicerceLabs](https://alicercelabs.com.br): IP, CEP, DNS, email, filas, banco de dados edge, execução de WASM e o resto das 16 APIs, todas atrás do mesmo formato de resposta. As de consulta pura (IP, CEP, DNS, email, SSL, confiabilidade, mapas, QR code, imagem, fatura) respondem sem nenhuma credencial: `New("")` já funciona, numa cota menor. Pra cota maior nessas, ou pra usar as que guardam dado seu (chave-valor, fila, banco edge, funções, cron, uptime, que sempre exigem um token), é só registrar, ver "Ainda não tem uma chave?" abaixo.

Zero dependências externas, só a standard library.

```bash
go get github.com/alicercelabs/alicercesdk-go
```

## Início rápido

```go
package main

import (
	"context"
	"fmt"

	alicercelabs "github.com/alicercelabs/alicercesdk-go"
)

func main() {
	client := alicercelabs.New("alk_...") // ou um token JWT de login/register

	endereco, err := client.CEP.Get(context.Background(), "01310100", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(endereco.Logradouro) // "Avenida Paulista"
}
```

Ainda não tem uma chave? Nas APIs de consulta pura, `New("")` já funciona sem mais nada, numa cota menor por IP (100/dia em vez de 1.000/dia). `Register`/`Login` guardam o token no client sozinhos, se você quiser a cota maior ou uma das APIs que guardam dado seu (KV, Queue, EdgeDB, Functions, Cron, UpTime, essas sempre exigem token):

```go
client := alicercelabs.New("")
_, err := client.Auth.Register(ctx, "voce", "voce@exemplo.com", "alguma-coisa-forte")
client.CEP.Get(ctx, "01310100", nil) // já autenticado
```

## O que tem aqui

Um campo por API, todos no mesmo client:

| Campo | API |
|---|---|
| `client.IP` | IP Intelligence — geo, ASN, cloud, VPN/Tor (`Lookup`/`Self`/`Batch`) |
| `client.CEP` | Endereço a partir do CEP |
| `client.CNPJ` | Consulta de CNPJ (fonte local + fallback BrasilAPI) |
| `client.CPF` | Validação de CPF e região fiscal |
| `client.Feriados` | Feriados nacionais por ano |
| `client.DiasUteis` | Contagem de dias úteis num intervalo |
| `client.ISBN` | Metadados de livro por ISBN |
| `client.IBGE` | Regiões, estados, municípios e classes CNAE |
| `client.Bancos` | Lista de bancos (participantes do STR) |
| `client.NCM` | Nomenclatura Comum do Mercosul |
| `client.OMS` | CID-10 |
| `client.Cambio` | Cotação de câmbio (PTAX) |
| `client.Taxas` | Taxas e índices oficiais (Selic, CDI, IPCA, IGP-M) |
| `client.RegistroBR` | Disponibilidade de domínio .br |
| `client.PIX` | Participantes do PIX |
| `client.DNS` | Consulta DNS |
| `client.Email` | Verificação de email |
| `client.SSL` | Checagem de certificado |
| `client.Maps` | Geocodificação e rotas |
| `client.Trust` | Score de confiabilidade |
| `client.KV` | Armazenamento chave-valor |
| `client.Queue` | Filas FIFO |
| `client.EdgeDB` | Banco de dados edge (SQLite por cliente) |
| `client.Cron` | Jobs agendados |
| `client.UpTime` | Monitoramento de uptime |
| `client.QRCode` | Gerador de QR Code |
| `client.Imagem` | Transformação de imagem |
| `client.Templating` | Geração de fatura em PDF |
| `client.Functions` | Execução de WASM |
| `client.Auth` | Registro, login, perfil |
| `client.Account` | Suas próprias API keys e analytics de uso |

Cada parâmetro e cada campo de resposta está documentado em [alicercelabs.com.br](https://alicercelabs.com.br).

## Exemplos

**Ver seus próprios dados de conta e uso:**

```go
me, _ := client.Auth.Me(ctx)
fmt.Println(me.Username, me.Email)

uso, _ := client.Account.Usage(ctx, 7) // últimos 7 dias
for _, linha := range uso {
	fmt.Println(linha.API, linha.Operation, linha.RequestCount)
}
```

**Gerenciar API keys:**

```go
nova, _ := client.Account.APIKeys.Create(ctx, "ci-pipeline")
fmt.Println(nova.Key) // só aparece aqui, salve agora

keys, _ := client.Account.APIKeys.List(ctx)
for _, k := range keys {
	fmt.Println(k.Name, k.Active)
}
```

**KV, Queue, Edge DB:**

```go
client.KV.Put(ctx, "tema", "escuro", 3600)
valor, _ := client.KV.Get(ctx, "tema") // "escuro"

client.Queue.Push(ctx, "pedidos", "pedido-123")
msg, ok, _ := client.Queue.Pull(ctx, "pedidos", 3) // long-poll de até 3s

client.EdgeDB.Query(ctx, "meubanco", "CREATE TABLE t (id INTEGER PRIMARY KEY, nome TEXT)", nil)
client.EdgeDB.Query(ctx, "meubanco", "INSERT INTO t (nome) VALUES (?)", []any{"Fulano"})
resultado, _ := client.EdgeDB.Query(ctx, "meubanco", "SELECT * FROM t", nil)
fmt.Println(resultado.Rows)
```

**Endpoints que devolvem arquivo** (QRCode, Imagem, Templating, Functions `Invoke`) devolvem um `*BinaryResponse`, não o envelope JSON de sempre:

```go
qr, _ := client.QRCode.Generate(ctx, "https://alicercelabs.com.br", 512)
qr.Save("qrcode.png")

pix, _ := client.QRCode.Pix(ctx, alicercelabs.PixParams{
	Chave:  "11999999999",
	Nome:   "Fulano de Tal",
	Cidade: "Sao Paulo",
	Valor:  10.50, // opcional, sem isso quem paga digita o valor
})
pix.Save("pix.png")
fmt.Println(pix.CopiaCola) // o mesmo payload em texto, pra quem quiser mostrar o "copia e cola"

fatura, _ := client.Templating.Invoice(ctx, alicercelabs.InvoiceRequest{
	Issuer:    alicercelabs.InvoiceParty{Name: "Minha Empresa"},
	Recipient: alicercelabs.InvoiceParty{Name: "Cliente Exemplo"},
	Items: []alicercelabs.InvoiceItem{
		{Description: "Consultoria", Quantity: 2, UnitPrice: 500},
	},
})
fatura.Save("fatura.pdf")
```

**Functions** (WASM, qualquer linguagem que compile pra WASI, incluindo o próprio Go):

```go
wasm, _ := os.ReadFile("minha_funcao.wasm")
client.Functions.Deploy(ctx, "minha", wasm)

resposta, _ := client.Functions.Invoke(ctx, "minha", []byte("algum corpo"))
fmt.Println(string(resposta.Content))
```

## Erros

Toda chamada com falha devolve um `*alicercelabs.APIError`:

```go
endereco, err := client.CEP.Get(ctx, "00000000", nil)
if err != nil {
	var apiErr *alicercelabs.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("erro %d: %s\n", apiErr.StatusCode, apiErr.Message)
	}
}
```

Ou use os helpers de conveniência pra checar o status sem desempacotar o erro você mesmo:

```go
if alicercelabs.IsNotFound(err) { ... }
if alicercelabs.IsRateLimit(err) {
	var apiErr *alicercelabs.APIError
	errors.As(err, &apiErr)
	fmt.Println("espera", apiErr.RetryAfter, "segundos")
}
```

| Helper | Status |
|---|---|
| `IsValidationError` | 400 |
| `IsAuthenticationError` | 401 |
| `IsNotFound` | 404 |
| `IsRateLimit` | 429 (`APIError.RetryAfter`, em segundos) |
| `IsServiceUnavailable` | 503 |

## Configuração avançada

```go
client := alicercelabs.New(
	"alk_...",
	alicercelabs.WithAPIBase("https://api.alicercelabs.com.br"),      // padrão
	alicercelabs.WithAccountBase("https://app.alicercelabs.com.br"),  // padrão, só client.Account.*
	alicercelabs.WithTimeout(30*time.Second),
)
```

`AccountBase` existe separado de `APIBase` porque API keys e analytics de uso vivem no backend do painel, não no host das APIs de produto. Não precisa pensar nisso no dia a dia, o SDK já manda cada chamada pro host certo.

## Desenvolvimento

```bash
go build ./...
go vet ./...
go test ./...              # unitários, ~90% de cobertura de statements
go test -cover ./...       # com o número de cobertura
```

Os testes unitários sobem um `httptest.Server` real e batem nele, um teste por método de API (`resources_test.go`), mais a máquina de request/erro em si (`client_test.go`) e as ramificações de erro (`errorpaths_test.go`, `coverage_test.go`).

### Testes de integração

`integration_test.go` bate numa instância real da AlicerceLabs (produção por padrão), usando a mesma API pública que qualquer chamador usaria. Ele registra uma conta descartável de verdade, cria e apaga recursos de verdade (chaves KV, uma fila, um Edge DB, um job de cron, um monitor de uptime, uma função, uma API key) e no fim apaga a própria conta. Por isso é opt-in, atrás de uma build tag:

```bash
ALICERCELABS_INTEGRATION=1 go test -tags integration -run TestIntegration -v .
```

`ALICERCELABS_API_BASE`/`ALICERCELABS_ACCOUNT_BASE` apontam pra uma instância self-hosted em vez de produção, se for o caso.

Não testado de propósito: `Cron.WorkerStart`/`WorkerStop` e `UpTime.WorkerStart`/`WorkerStop`. Esses controlam um daemon compartilhado por toda a instância, não algo isolado à conta de teste, então pará-lo aqui afetaria usuários de verdade. `WorkerStatus` (só leitura) é testado no lugar deles.

## Licença

MIT
