# Crawler — Lenovo Laptops (webscraper.io)

Crawler em JavaScript/Node.js que coleta todos os notebooks da marca **Lenovo** do site [webscraper.io](https://webscraper.io/test-sites/e-commerce/static/computers/laptops), ordenados do mais barato para o mais caro.

---

## Como executar

1. Instale as dependências:

```bash
npm install
```

2. Configure as variáveis de ambiente:

```bash
cp .env.example .env
```

3. Execute o crawler:

```bash
node crawler.js
```

O resultado é impresso no terminal e salvo no arquivo definido em `OUTPUT_FILE` (padrão: `results.json`).

### Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `BASE_URL` | URL do site | URL base do crawler |
| `BRAND_KEYWORD` | `lenovo` | Marca a filtrar |
| `REQUEST_TIMEOUT_MS` | `15000` | Timeout por requisição (ms) |
| `RATE_LIMIT_MS` | `1000` | Pausa entre páginas (ms) |
| `MAX_RETRIES` | `3` | Tentativas em erro transitório |
| `OUTPUT_FILE` | `results.json` | Arquivo de saída |

### Requisitos

- Node.js 18 ou superior (necessário para `fetch` nativo e `AbortSignal.timeout`)

### Dependências

```
cheerio ^1.0.0
```

---

## Decisões técnicas

### 1. `cheerio` no lugar de jQuery

jQuery é uma biblioteca de browser e não roda nativamente em Node.js. `cheerio` oferece a mesma API (`.find()`, `.attr()`, `.text()`, `.each()`, `.not()`) sem depender de um DOM real, sendo a escolha padrão para scraping em Node.js.

### 2. `fetch` nativo + `AbortSignal.timeout`

Disponível a partir do Node.js 18, elimina dependências externas como `axios` ou `node-fetch`. O timeout de 15 segundos evita que requisições travem o processo indefinidamente em caso de rede instável.

### 3. Verificação explícita de status HTTP

Sem essa verificação, um erro 429 ou 500 retornaria HTML de página de erro que o parser aceitaria silenciosamente — resultando em coleta vazia sem nenhum aviso.

### 4. Retry com backoff exponencial

Erros transitórios (429 e 5xx) são retentados até 3 vezes com espera crescente de 2s e 4s entre tentativas. Erros não-transitórios (ex: 404) não são retentados, evitando loops desnecessários.

### 5. Rate limiting entre páginas

Um `sleep(1000)` entre requisições evita disparar todas as 20 páginas em sequência máxima, reduzindo o risco de bloqueio por rate limiting do servidor.

### 6. Filtro somente por `'lenovo'`

Palavras-chave como `'thinkpad'` e `'yoga'` foram descartadas. Alguns produtos antigos no site usam essas sub-marcas sem o prefixo "Lenovo" — incluí-los seria um falso positivo, pois o site não os identifica como produtos Lenovo.

### 7. Atributo `title` para nome completo

O CSS do site trunca visualmente títulos longos. O atributo `title=""` da tag `<a>` sempre contém o nome completo, sem truncamento.

### 8. Ordenação global por preço

A ordenação é aplicada após coletar todas as páginas, garantindo que o resultado final seja globalmente ordenado e não apenas por página.

---

## Campos coletados

| Campo         | Descrição                   |
| ------------- | --------------------------- |
| `name`        | Nome completo do produto    |
| `price`       | Preço em dólares (float)    |
| `description` | Especificações técnicas     |
| `rating`      | Avaliação em estrelas (0–5) |
| `reviews`     | Número de avaliações        |
| `link`        | URL da página do produto    |
