package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"zedsellauto/internal/config"
	"zedsellauto/internal/database"
	"zedsellauto/internal/marketdata"
	"zedsellauto/internal/repository"
)

func main() {
	maxPages := flag.Int("pages", 0, "maximum pages to import; 0 imports every detected page")
	delayMS := flag.Int("delay-ms", 250, "delay between page requests")
	dryRun := flag.Bool("dry-run", false, "fetch and parse without writing to PostgreSQL")
	flag.Parse()

	config.LoadDotEnv(".env")
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 30 * time.Second}
	prices, err := marketdata.FetchKrungsriUsedCarPrices(ctx, client, marketdata.KrungsriOptions{
		Delay:    time.Duration(*delayMS) * time.Millisecond,
		MaxPages: *maxPages,
	})
	if err != nil {
		log.Fatalf("fetch krungsri prices: %v", err)
	}

	if *dryRun {
		fmt.Printf("parsed %d Krungsri used-car prices\n", len(prices))
		return
	}

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate postgres: %v", err)
	}

	repo := repository.New(db)
	written, err := repo.UpsertMarketUsedCarPrices(ctx, prices)
	if err != nil {
		log.Fatalf("upsert prices: %v", err)
	}

	total, err := repo.CountMarketUsedCarPrices(ctx)
	if err != nil {
		log.Fatalf("count imported prices: %v", err)
	}

	fmt.Printf("imported %d Krungsri used-car prices, changed %d rows, table total %d\n", len(prices), written, total)
}
