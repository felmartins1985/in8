package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "strings"
    "time"

    "golang.org/x/net/html"
)

const (
    baseURL      = "http://books.toscrape.com/catalogue/"
    startPage    = "page-1.htm"
    requestDelay = 1 * time.Second
    httpTimeout  = 15 * time.Second
    maxRetries   = 3
)

type Book struct {
    Title  string
    Price  string
    Rating string
}

type Crawler struct {
    client *http.Client
    books  []Book
}

func NewCrawler() *Crawler {
    return &Crawler{
        client: &http.Client{
            Timeout: httpTimeout,
        },
    }
}

func (c *Crawler) tryFetch(ctx context.Context, url string) (*html.Node, bool, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, false, err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, true, err
    }
    defer resp.Body.Close()

    switch {
    case resp.StatusCode == http.StatusTooManyRequests:
        return nil, true, fmt.Errorf("rate limited (429) em %s", url)
    case resp.StatusCode >= 500:
        return nil, true, fmt.Errorf("erro do servidor (%d) em %s", resp.StatusCode, url)
    case resp.StatusCode != http.StatusOK:
        return nil, false, fmt.Errorf("status inesperado %d em %s", resp.StatusCode, url)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, true, err
    }

    doc, err := html.Parse(strings.NewReader(string(body)))
    if err != nil {
        return nil, false, err
    }

    return doc, false, nil
}

func (c *Crawler) fetchPage(ctx context.Context, url string) (*html.Node, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            backoff := time.Duration(attempt) * 2 * time.Second
            log.Printf("tentativa %d/%d, aguardando %v antes de tentar novamente...", attempt+1, maxRetries, backoff)
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        doc, retry, err := c.tryFetch(ctx, url)
        if err == nil {
            return doc, nil
        }

        lastErr = err
        log.Printf("erro ao buscar %s: %v", url, err)

        if !retry {
            return nil, err
        }
    }

    return nil, fmt.Errorf("todas as %d tentativas falharam para %s: %w", maxRetries, url, lastErr)
}

func (c *Crawler) parseBooks(doc *html.Node) {
    var walk func(*html.Node)
    walk = func(n *html.Node) {
        if n.Type == html.ElementNode && hasClass(n, "product_pod") {
            c.books = append(c.books, extractBook(n))
        }
        for child := n.FirstChild; child != nil; child = child.NextSibling {
            walk(child)
        }
    }
    walk(doc)
}

func extractBook(n *html.Node) Book {
    var title, price, rating string
    var walk func(*html.Node)
    walk = func(n *html.Node) {
        if n.Type == html.ElementNode {
            switch n.Data {
            case "p":
                if hasClass(n, "star-rating") {
                    for _, a := range n.Attr {
                        if a.Key == "class" {
                            parts := strings.Fields(a.Val)
                            if len(parts) > 1 {
                                rating = parts[1]
                            }
                        }
                    }
                }
                if hasClass(n, "price_color") && n.FirstChild != nil {
                    price = n.FirstChild.Data
                }
            case "a":
                for _, a := range n.Attr {
                    if a.Key == "title" {
                        title = a.Val
                    }
                }
            }
        }
        for child := n.FirstChild; child != nil; child = child.NextSibling {
            walk(child)
        }
    }
    walk(n)
    return Book{title, price, rating}
}

func getNextPage(doc *html.Node) string {
    var nextPage string
    var walk func(*html.Node)
    walk = func(n *html.Node) {
        if nextPage != "" {
            return
        }
        if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "next") {
            for child := n.FirstChild; child != nil; child = child.NextSibling {
                if child.Type == html.ElementNode && child.Data == "a" {
                    for _, attr := range child.Attr {
                        if attr.Key == "href" {
                            nextPage = attr.Val
                            return
                        }
                    }
                }
            }
        }
        for child := n.FirstChild; child != nil; child = child.NextSibling {
            walk(child)
        }
    }
    walk(doc)
    return nextPage
}

func hasClass(n *html.Node, className string) bool {
    for _, attr := range n.Attr {
        if attr.Key == "class" {
            for _, class := range strings.Fields(attr.Val) {
                if class == className {
                    return true
                }
            }
        }
    }
    return false
}

func (c *Crawler) crawl(ctx context.Context) error {
    page := startPage

    for page != "" {
        url := baseURL + page
        log.Printf("coletando página: %s", url)

        doc, err := c.fetchPage(ctx, url)
        if err != nil {
            return fmt.Errorf("erro ao buscar página %s: %w", url, err)
        }

        c.parseBooks(doc)
        page = getNextPage(doc)

        if page != "" {
            time.Sleep(requestDelay)
        }
    }

    return nil
}

func (c *Crawler) printBooks() {
    for _, book := range c.books {
        fmt.Printf("Título: %s | Preço: %s | Avaliação: %s\n", book.Title, book.Price, book.Rating)
    }
}

func main() {
    ctx := context.Background()

    crawler := NewCrawler()

    if err := crawler.crawl(ctx); err != nil {
        log.Fatalf("erro durante execução do crawler: %v", err)
    }

    crawler.printBooks()
}