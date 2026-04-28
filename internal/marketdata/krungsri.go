package marketdata

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"zedsellauto/internal/domain"
)

const krungsriUsedCarWarehouseURL = "https://krungsrimarket.cjdataservice.com/usedcar/warehouse"

var (
	krungsriMaxPagePattern = regexp.MustCompile(`warehouse\?page=(\d+)`)
	krungsriRowPattern     = regexp.MustCompile(`(?is)<tr\s+class="clickable-row"\s+data-href="([^"]+)">\s*<td>.*?</td>\s*<td>(.*?)</td>\s*<td>(.*?)</td>\s*<td>(.*?)</td>\s*<td>(.*?)</td>`)
	krungsriTagPattern     = regexp.MustCompile(`(?is)<[^>]+>`)
	krungsriModelDate      = regexp.MustCompile(`-(\d{4})(\d{2})\s*$`)
)

type KrungsriOptions struct {
	Delay    time.Duration
	MaxPages int
}

func FetchKrungsriUsedCarPrices(ctx context.Context, client *http.Client, options KrungsriOptions) ([]domain.MarketUsedCarPrice, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if options.Delay <= 0 {
		options.Delay = 250 * time.Millisecond
	}

	firstPage, err := fetchKrungsriPage(ctx, client, 1)
	if err != nil {
		return nil, err
	}

	maxPage := detectKrungsriMaxPage(firstPage)
	if maxPage == 0 {
		maxPage = 1
	}
	if options.MaxPages > 0 && options.MaxPages < maxPage {
		maxPage = options.MaxPages
	}

	prices := parseKrungsriPrices(firstPage)
	seen := map[string]struct{}{}
	for _, price := range prices {
		seen[price.SourceURL] = struct{}{}
	}

	for page := 2; page <= maxPage; page++ {
		select {
		case <-ctx.Done():
			return prices, ctx.Err()
		case <-time.After(options.Delay):
		}

		body, err := fetchKrungsriPage(ctx, client, page)
		if err != nil {
			return prices, err
		}
		for _, price := range parseKrungsriPrices(body) {
			if _, exists := seen[price.SourceURL]; exists {
				continue
			}
			seen[price.SourceURL] = struct{}{}
			prices = append(prices, price)
		}
	}

	return prices, nil
}

func fetchKrungsriPage(ctx context.Context, client *http.Client, page int) (string, error) {
	url := krungsriUsedCarWarehouseURL
	if page > 1 {
		url = fmt.Sprintf("%s?page=%d", krungsriUsedCarWarehouseURL, page)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "ZedSellAuto market price importer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("krungsri page %d returned status %d", page, resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func detectKrungsriMaxPage(body string) int {
	matches := krungsriMaxPagePattern.FindAllStringSubmatch(body, -1)
	maxPage := 0
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		page, err := strconv.Atoi(match[1])
		if err == nil && page > maxPage {
			maxPage = page
		}
	}
	return maxPage
}

func parseKrungsriPrices(body string) []domain.MarketUsedCarPrice {
	matches := krungsriRowPattern.FindAllStringSubmatch(body, -1)
	prices := make([]domain.MarketUsedCarPrice, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		sourceURL := strings.TrimSpace(html.UnescapeString(match[1]))
		brand := cleanCell(match[2])
		rawModel := cleanCell(match[3])
		monthlyPayment := parseInt(cleanCell(match[4]))
		priceMin, priceMax := parsePriceRange(cleanCell(match[5]))
		model, modelYear, modelMonth := splitKrungsriModel(rawModel)

		if sourceURL == "" || brand == "" || model == "" || modelYear == 0 || priceMin == 0 || priceMax == 0 {
			continue
		}

		prices = append(prices, domain.MarketUsedCarPrice{
			Source:            "krungsri_market",
			SourceURL:         sourceURL,
			Brand:             brand,
			Model:             model,
			RawModel:          rawModel,
			ModelYear:         modelYear,
			ModelMonth:        modelMonth,
			MonthlyPaymentTHB: monthlyPayment,
			PriceMinTHB:       priceMin,
			PriceMaxTHB:       priceMax,
		})
	}
	return prices
}

func cleanCell(value string) string {
	withoutTags := krungsriTagPattern.ReplaceAllString(value, "")
	decoded := html.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(decoded), " ")
}

func splitKrungsriModel(rawModel string) (string, int, int) {
	match := krungsriModelDate.FindStringSubmatch(rawModel)
	if len(match) != 3 {
		return strings.TrimSpace(rawModel), 0, 0
	}

	year, _ := strconv.Atoi(match[1])
	month, _ := strconv.Atoi(match[2])
	model := strings.TrimSpace(strings.TrimSuffix(rawModel, "-"+match[1]+match[2]))
	return model, year, month
}

func parsePriceRange(value string) (int64, int64) {
	parts := strings.Split(value, "-")
	if len(parts) == 1 {
		price := int64(parseInt(parts[0]))
		return price, price
	}

	minPrice := int64(parseInt(parts[0]))
	maxPrice := int64(parseInt(parts[len(parts)-1]))
	return minPrice, maxPrice
}

func parseInt(value string) int {
	normalized := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
	parsed, err := strconv.Atoi(normalized)
	if err != nil {
		return 0
	}
	return parsed
}
