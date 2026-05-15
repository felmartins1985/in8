import * as cheerio from 'cheerio';
import { writeFileSync } from 'fs';

const BASE_URL = 'https://webscraper.io/test-sites/e-commerce/static/computers/laptops';
const LENOVO_KEYWORDS = ['lenovo'];

const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

function isLenovo({ name, description }) {
  const text = `${name} ${description}`.toLowerCase();
  return LENOVO_KEYWORDS.some(kw => text.includes(kw));
}

async function fetchPage(url, attempt = 1) {
  const res = await fetch(url, {
    signal: AbortSignal.timeout(15_000),
    headers: { 'Accept': 'text/html' },
  });

  if (res.status === 200) return res.text();

  if ((res.status === 429 || res.status >= 500) && attempt < 3) {
    const delay = 2000 * attempt;
    console.warn(`  HTTP ${res.status} — retrying in ${delay}ms (attempt ${attempt}/3)`);
    await sleep(delay);
    return fetchPage(url, attempt + 1);
  }

  throw new Error(`HTTP ${res.status}: ${url}`);
}

function getTotalPages(html) {
  const $ = cheerio.load(html);
  let max = 1;
  $('.pagination a').each((_, el) => {
    const n = parseInt($(el).text(), 10);
    if (!isNaN(n) && n > max) max = n;
  });
  return max;
}

function parsePage(html) {
  const $ = cheerio.load(html);
  const products = [];

  $('.thumbnail').each((_, el) => {
    const $card = $(el);

    const $titleEl = $card.find('a.title');
    const name = $titleEl.attr('title') || $titleEl.text().trim();

    const href = $titleEl.attr('href') || '';
    const link = href.startsWith('http') ? href : `https://webscraper.io${href}`;

    const price = parseFloat(
      $card.find('.price').first().text().replace(/[^0-9.]/g, '')
    );

    const description = $card.find('.description').text().trim();

    const rating = $card
      .find('.ratings .glyphicon-star')
      .not('.glyphicon-star-empty')
      .length;

    const reviews = parseInt($card.find('.ratings .pull-right').text(), 10) || 0;

    const imgSrc = $card.find('img').attr('src') || '';
    const imageUrl = imgSrc.startsWith('http') ? imgSrc : `https://webscraper.io${imgSrc}`;

    products.push({ name, price, description, rating, reviews, link, imageUrl });
  });

  return products;
}

async function crawl() {
  console.log('Fetching page 1...');
  const firstHtml = await fetchPage(BASE_URL);
  const totalPages = getTotalPages(firstHtml);
  console.log(`Total pages: ${totalPages}\n`);

  const results = parsePage(firstHtml).filter(isLenovo);

  for (let page = 2; page <= totalPages; page++) {
    await sleep(1000);
    console.log(`Fetching page ${page}/${totalPages}...`);
    const html = await fetchPage(`${BASE_URL}?page=${page}`);
    parsePage(html).filter(isLenovo).forEach(p => results.push(p));
  }

  results.sort((a, b) => a.price - b.price);

  console.log(`\n${'─'.repeat(60)}`);
  console.log(`${results.length} Lenovo laptops encontrados (mais barato → mais caro)`);
  console.log(`${'─'.repeat(60)}\n`);

  results.forEach((p, i) => {
    const stars = '★'.repeat(p.rating) + '☆'.repeat(5 - p.rating);
    console.log(`${String(i + 1).padStart(2)}. ${p.name}`);
    console.log(`    Preço:     $${p.price.toFixed(2)}`);
    console.log(`    Descrição: ${p.description}`);
    console.log(`    Rating:    ${stars} (${p.rating}/5)`);
    console.log(`    Reviews:   ${p.reviews}`);
    console.log(`    Link:      ${p.link}`);
    console.log();
  });

  writeFileSync('results.json', JSON.stringify(results, null, 2), 'utf8');
  console.log('Resultados salvos em results.json');
}

crawl().catch(err => {
  console.error('Crawler falhou:', err.message);
  process.exit(1);
});