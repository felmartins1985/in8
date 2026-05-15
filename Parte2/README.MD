# Crawler — books.toscrape.com

Crawler em Go que coleta título, preço e avaliação de todos os livros do site [books.toscrape.com](http://books.toscrape.com).

---

## Como executar

```bash
go mod tidy
go run crawler.go
```

### Dependências

```
golang.org/x/net
```

Usada para parsing de HTML (`golang.org/x/net/html`). É a biblioteca padrão de fato para essa tarefa em Go — mantida pelo próprio time da linguagem, robusta e amplamente adotada.

---

## Problemas encontrados no código original

### 1. Estado global mutável

**Código original:**
```go
var results []Book
```

**Impacto em produção:** Em qualquer cenário com goroutines ou múltiplas execuções no mesmo processo, o estado compartilhado causa race conditions e resultados imprevisíveis.

**Correção:** Estado encapsulado na struct `Crawler`, eliminando dependência de variável global.

---

### 2. HTTP client sem timeout

**Código original:**
```go
resp, err := http.DefaultClient.Do(req)
```

**Impacto em produção:** `http.DefaultClient` não tem timeout definido. Em rede instável ou servidor lento, a goroutine trava indefinidamente, travando todo o processo.

**Correção:**
```go
client: &http.Client{Timeout: 15 * time.Second}
```

---

### 3. Sem verificação do status HTTP

**Código original:** Nenhuma verificação de `resp.StatusCode`.

**Impacto em produção:** Um 429 (rate limit) ou 500 (erro do servidor) retorna HTML de página de erro. O parser aceita o HTML silenciosamente, resultando em coleta de zero livros sem nenhum aviso.

**Correção:** `switch` explícito tratando 200, 429, 5xx e outros status.

---

### 4. Sem retry para erros transitórios

**Código original:**
```go
doc, err := fetchPage(base + page)
if err != nil {
    log.Println("Erro:", err)
    break
}
```

**Impacto em produção:** Qualquer falha de rede momentânea ou 429/500 aborta toda a coleta. Em produção, um site como esse aplicará rate limiting com frequência.

**Correção:** Retry com backoff exponencial (até 3 tentativas, com espera de 2s e 4s entre elas). Erros não-transitórios (ex: 404) não são retentados.

---

### 5. `Accept-Encoding: zstd` sem decompressão

**Código original:**
```go
req.Header.Set("Accept-Encoding", "zstd")
```

**Impacto em produção:** O Go descomprime `gzip` e `deflate` automaticamente, mas **não descomprime `zstd`**. Se o servidor respeitar o header, o body retornado é binário comprimido. O parser HTML tentará interpretar bytes corrompidos, produzindo resultados vazios ou erro silencioso.

**Correção:** Header removido. O Go negocia automaticamente encodings que sabe descomprimir.

---

### 6. Double `parseBooks` na última página

**Código original:**
```go
next := getNextPage(doc)
if next == "" {
    parseBooks(doc) // chamada duplicada
}
page = next
```

**Impacto em produção:** Todos os livros da última página são inseridos duas vezes nos resultados.

**Correção:** Bloco removido. `parseBooks` já é chamado antes de verificar a próxima página.

---

### 7. Pula a primeira página

**Código original:**
```go
page := "page-2.htm"
```

**Impacto em produção:** Os 20 primeiros livros nunca são coletados.

**Correção:** `startPage = "page-1.htm"`.

---

### 8. `hasClass` com falso positivo

**Código original:**
```go
strings.Contains(a.Val, class)
```

**Impacto em produção:** `hasClass(n, "price")` retornaria `true` para um elemento com classe `"price_color"`. Dependendo de futuras mudanças no HTML do site, isso pode parsear elementos errados silenciosamente.

**Correção:** `strings.Fields` divide as classes e compara por igualdade exata.

---

### 9. Sem `context.Context`

**Impacto em produção:** Impossível cancelar a execução externamente (ex: timeout global, sinal de shutdown, integração com outros sistemas).

**Correção:** `context.Context` propagado por toda a cadeia de chamadas.

---

### 10. Sem rate limiting

**Impacto em produção:** Sem pausa entre requisições, o crawler dispara todas as páginas em sequência máxima, aumentando a chance de ser bloqueado por rate limiting do servidor.

**Correção:** `time.Sleep(1 * time.Second)` entre páginas (aplicado apenas quando há próxima página).

---

## Resumo das correções

| Problema | Severidade | Impacto |
|---|---|---|
| HTTP client sem timeout | Alta | Travamento indefinido |
| Sem verificação de status HTTP | Alta | Coleta silenciosa de dados corrompidos |
| Sem retry | Alta | Falha total em qualquer erro transitório |
| `Accept-Encoding: zstd` | Alta | Body corrompido em produção |
| Estado global | Média | Race condition com goroutines |
| Double `parseBooks` | Média | Duplicação de dados |
| Primeira página pulada | Média | Dados incompletos |
| `hasClass` falso positivo | Baixa | Parse incorreto em mudanças futuras do HTML |
| Sem context | Baixa | Impossível cancelar execução |
| Sem rate limiting | Baixa | Risco de bloqueio por 429 |