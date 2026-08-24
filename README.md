# alicercesdk-go

SDK oficial em Go para a [AlicerceLabs](https://alicercelabs.com.br) — infra básica de API pra quem constrói no Brasil (IP, CEP, DNS, email, filas, banco de dados edge, execução de WASM e mais, todas atrás de uma autenticação e um formato de resposta só).

Zero dependências externas — só a standard library.

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

Ainda não tem uma chave? `Register`/`Login` guardam o token no client sozinhos:

```go
client := alicercelabs.New("")
_, err := client.Auth.Register(ctx, "voce", "voce@exemplo.com", "alguma-coisa-forte")
client.CEP.Get(ctx, "01310100", nil) // já autenticado
```

## O que tem aqui

Um campo por API, todos no mesmo client:

| Campo | API |
|---|---|
| `client.IP` | Geolocalização de IP |
| `client.CEP` | Endereço a partir do CEP |
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

Documentação completa de cada API, com todos os parâmetros: [alicercelabs.com.br](https://alicercelabs.com.br).

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
fmt.Println(nova.Key) // só aparece aqui — salve agora

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

**Endpoints que devolvem arquivo** (QRCode, Imagem, Templating, Functions `Invoke`) devolvem um `*BinaryResponse`:

```go
qr, _ := client.QRCode.Generate(ctx, "https://alicercelabs.com.br", 512)
qr.Save("qrcode.png")

fatura, _ := client.Templating.Invoice(ctx, alicercelabs.InvoiceRequest{
	Issuer:    alicercelabs.InvoiceParty{Name: "Minha Empresa"},
	Recipient: alicercelabs.InvoiceParty{Name: "Cliente Exemplo"},
	Items: []alicercelabs.InvoiceItem{
		{Description: "Consultoria", Quantity: 2, UnitPrice: 500},
	},
})
fatura.Save("fatura.pdf")
```

**Functions** (WASM, qualquer linguagem que compile pra WASI — incluindo o próprio Go):

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
	alicercelabs.WithAccountBase("https://app.alicercelabs.com.br"),  // padrão — só client.Account.*
	alicercelabs.WithTimeout(30*time.Second),
)
```

## Desenvolvimento

```bash
go build ./...
go vet ./...
go test ./...
```

## Licença

MIT
