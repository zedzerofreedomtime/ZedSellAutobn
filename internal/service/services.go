package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"zedsellauto/internal/config"
	"zedsellauto/internal/domain"
	"zedsellauto/internal/repository"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
)

type Services struct {
	cfg   config.Config
	repo  *repository.Repository
	cache *redis.Client
}

func NewServices(cfg config.Config, db *pgxpool.Pool, cache *redis.Client) *Services {
	return &Services{
		cfg:   cfg,
		repo:  repository.New(db),
		cache: cache,
	}
}

func (s *Services) Signup(ctx context.Context, email, password, fullName string) (domain.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", err
	}

	user, err := s.repo.CreateUser(ctx, email, string(hash), fullName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, "", ErrConflict
		}
		return domain.User{}, "", err
	}

	token, err := s.signToken(user.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, token, nil
}

func (s *Services) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	user, hash, err := s.repo.FindUserAuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.User{}, "", ErrUnauthorized
		}
		return domain.User{}, "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return domain.User{}, "", ErrUnauthorized
	}

	token, err := s.signToken(user.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, token, nil
}

func (s *Services) Me(ctx context.Context, token string) (domain.User, error) {
	userID, err := s.parseToken(token)
	if err != nil {
		return domain.User{}, ErrUnauthorized
	}
	return s.repo.FindUserByID(ctx, userID)
}

func (s *Services) Home(ctx context.Context) (domain.HomePayload, error) {
	featuredVehicles, err := s.repo.ListVehicles(ctx, "", true, 4)
	if err != nil {
		return domain.HomePayload{}, err
	}
	categories, err := s.repo.ListVehicleCategories(ctx)
	if err != nil {
		return domain.HomePayload{}, err
	}
	posts, err := s.repo.ListBlogPosts(ctx)
	if err != nil {
		return domain.HomePayload{}, err
	}

	var featuredPost *domain.BlogPost
	if len(posts) > 0 {
		featuredPost = &posts[0]
	}

	return domain.HomePayload{
		FeaturedVehicles: featuredVehicles,
		Categories:       categories,
		FeaturedPost:     featuredPost,
	}, nil
}

func (s *Services) VehicleCategories(ctx context.Context) ([]domain.VehicleCategory, error) {
	return s.repo.ListVehicleCategories(ctx)
}

func (s *Services) Vehicles(ctx context.Context, category string, featuredOnly bool) ([]domain.Vehicle, error) {
	cacheKey := fmt.Sprintf("vehicles:%s:%t", category, featuredOnly)
	if payload, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
		var cached []domain.Vehicle
		if json.Unmarshal([]byte(payload), &cached) == nil {
			return cached, nil
		}
	}

	if category == "all" {
		category = ""
	}

	items, err := s.repo.ListVehicles(ctx, category, featuredOnly, 0)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(items); err == nil {
		_ = s.cache.Set(ctx, cacheKey, encoded, time.Duration(s.cfg.ListingsCacheTTL)*time.Second).Err()
	}

	return items, nil
}

func (s *Services) VehicleDetail(ctx context.Context, slug string) (map[string]any, error) {
	vehicle, err := s.repo.GetVehicleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	vehicle.EstimatedMarketPrice = int64(float64(vehicle.PriceTHB) * 1.04)
	vehicle.NearbyListingCount = 128
	vehicle.AvgDaysOnMarket = 12

	related, err := s.repo.ListRelatedVehicles(ctx, slug, vehicle.CategorySlug, 3)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"vehicle":  vehicle,
		"related":  related,
		"services": []map[string]string{{"title": "Shipping"}, {"title": "Insurance"}, {"title": "Protection"}, {"title": "Inspection"}},
	}, nil
}

func (s *Services) BlogPosts(ctx context.Context) ([]domain.BlogPost, error) {
	return s.repo.ListBlogPosts(ctx)
}

func (s *Services) BlogDetail(ctx context.Context, slug string) (map[string]any, error) {
	post, err := s.repo.GetBlogPostBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	related, err := s.repo.ListRelatedBlogPosts(ctx, slug, post.Category, 3)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"post":    post,
		"related": related,
	}, nil
}

func (s *Services) Pricing(ctx context.Context) (map[string]any, error) {
	highlights, err := s.repo.ListPricingHighlights(ctx)
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.ListPricingPlans(ctx)
	if err != nil {
		return nil, err
	}
	faqs, err := s.repo.ListPricingFAQs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"highlights": highlights,
		"plans":      plans,
		"faqs":       faqs,
	}, nil
}

func (s *Services) HowItWorks(ctx context.Context) (map[string]any, error) {
	signals, err := s.repo.ListTrustSignals(ctx)
	if err != nil {
		return nil, err
	}
	steps, err := s.repo.ListHowItWorksSteps(ctx)
	if err != nil {
		return nil, err
	}
	buyer, err := s.repo.ListExperienceItems(ctx, "buyer")
	if err != nil {
		return nil, err
	}
	seller, err := s.repo.ListExperienceItems(ctx, "seller")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"trustSignals": signals,
		"steps":        steps,
		"buyer":        buyer,
		"seller":       seller,
	}, nil
}

func (s *Services) Favorites(ctx context.Context, token string) ([]domain.Vehicle, error) {
	userID, err := s.parseToken(token)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return s.repo.ListFavorites(ctx, userID)
}

func (s *Services) AddFavorite(ctx context.Context, token, vehicleID string) error {
	userID, err := s.parseToken(token)
	if err != nil {
		return ErrUnauthorized
	}
	if err := s.repo.AddFavorite(ctx, userID, vehicleID); err != nil {
		return err
	}
	return s.invalidateVehicleCache(ctx)
}

func (s *Services) RemoveFavorite(ctx context.Context, token, vehicleID string) error {
	userID, err := s.parseToken(token)
	if err != nil {
		return ErrUnauthorized
	}
	return s.repo.RemoveFavorite(ctx, userID, vehicleID)
}

func (s *Services) CreateOffer(ctx context.Context, token, vehicleID, fullName, email, phone string, amount int64, note string) error {
	userID, _ := s.optionalUserID(token)
	return s.repo.CreateOffer(ctx, vehicleID, userID, fullName, email, phone, amount, note)
}

func (s *Services) CreateTestDrive(ctx context.Context, token, vehicleID, fullName, email, phone string, preferredAt time.Time, note string) error {
	userID, _ := s.optionalUserID(token)
	return s.repo.CreateTestDrive(ctx, vehicleID, userID, fullName, email, phone, preferredAt, note)
}

func (s *Services) CreateInquiry(ctx context.Context, token, vehicleID, fullName, email, phone, message, channel string) error {
	userID, _ := s.optionalUserID(token)
	if channel == "" {
		channel = "chat"
	}
	return s.repo.CreateInquiry(ctx, vehicleID, userID, fullName, email, phone, message, channel)
}

func (s *Services) CreateFinanceApplication(ctx context.Context, token, vehicleID, fullName, email, phone string, downPercent float64, loanTerm int, creditBand string, income int64) error {
	userID, _ := s.optionalUserID(token)
	return s.repo.CreateFinanceApplication(ctx, vehicleID, userID, fullName, email, phone, downPercent, loanTerm, creditBand, income)
}

func (s *Services) CreateSellerVehicleSubmission(ctx context.Context, token string, input domain.SellerVehicleSubmissionInput) (domain.SellerVehicleSubmissionResult, error) {
	userID, _ := s.optionalUserID(token)
	result, err := s.repo.CreateSellerVehicleSubmission(ctx, userID, input)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}
	return result, s.invalidateVehicleCache(ctx)
}

func (s *Services) SellerValuations(ctx context.Context) ([]domain.ValuationRequest, error) {
	return s.repo.ListValuationRequests(ctx)
}

func (s *Services) CreateSellerValuation(ctx context.Context, token string, input domain.CreateValuationInput) (domain.ValuationRequest, error) {
	userID, _ := s.optionalUserID(token)
	preliminary := calculatePreliminaryAssessment(input.Vehicle)
	return s.repo.CreateValuationRequest(ctx, userID, input, preliminary)
}

func (s *Services) AddSellerValuationMessage(ctx context.Context, requestID, text string) (domain.ValuationRequest, error) {
	if strings.TrimSpace(text) == "" {
		return domain.ValuationRequest{}, repository.ErrNotFound
	}
	if _, err := s.repo.AddValuationMessage(ctx, requestID, "seller", strings.TrimSpace(text), nil); err != nil {
		return domain.ValuationRequest{}, err
	}
	return s.repo.GetValuationRequest(ctx, requestID)
}

func (s *Services) AddAdminValuationMessage(ctx context.Context, requestID, text string) (domain.ValuationRequest, error) {
	if strings.TrimSpace(text) == "" {
		return domain.ValuationRequest{}, repository.ErrNotFound
	}
	if _, err := s.repo.AddValuationMessage(ctx, requestID, "admin", strings.TrimSpace(text), nil); err != nil {
		return domain.ValuationRequest{}, err
	}
	return s.repo.GetValuationRequest(ctx, requestID)
}

func (s *Services) SendAdminValuationAssessment(ctx context.Context, requestID string, assessment domain.ValuationAssessment) (domain.ValuationRequest, error) {
	if assessment.EstimatedAt == "" {
		assessment.EstimatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := s.repo.SetValuationAssessment(ctx, requestID, assessment); err != nil {
		return domain.ValuationRequest{}, err
	}
	if _, err := s.repo.AddValuationMessage(ctx, requestID, "admin", buildAssessmentMessage(assessment), &assessment); err != nil {
		return domain.ValuationRequest{}, err
	}
	return s.repo.GetValuationRequest(ctx, requestID)
}

func (s *Services) PublishSellerValuation(ctx context.Context, requestID string, askingPriceTHB int64) (domain.ValuationRequest, error) {
	listing, err := s.repo.CreateSellerListingForValuation(ctx, requestID, askingPriceTHB)
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	message := fmt.Sprintf("Published %s at %s.", listing.Title, formatTHB(listing.PriceTHB))
	if _, err := s.repo.AddValuationMessage(ctx, requestID, "seller", message, nil); err != nil {
		return domain.ValuationRequest{}, err
	}
	if err := s.invalidateVehicleCache(ctx); err != nil {
		return domain.ValuationRequest{}, err
	}
	return s.repo.GetValuationRequest(ctx, requestID)
}

func (s *Services) SellerListings(ctx context.Context, category string) ([]domain.SellerListing, error) {
	if category == "all" {
		category = ""
	}
	return s.repo.ListSellerListings(ctx, category)
}

func (s *Services) SellerListingDetail(ctx context.Context, listingID string) (domain.SellerListing, error) {
	return s.repo.GetSellerListingByID(ctx, listingID)
}

func (s *Services) Health(ctx context.Context) map[string]string {
	return map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Services) signToken(userID string) (string, error) {
	exp := time.Now().Add(time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute).Unix()
	payload := fmt.Sprintf("%s|%d|%s", userID, exp, uuid.NewString())
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + signature
	return token, nil
}

func (s *Services) parseToken(token string) (string, error) {
	if token == "" {
		return "", ErrUnauthorized
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrUnauthorized
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write(payloadBytes)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", ErrUnauthorized
	}

	segments := strings.Split(string(payloadBytes), "|")
	if len(segments) < 2 {
		return "", ErrUnauthorized
	}
	expiry, err := strconv.ParseInt(segments[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return "", ErrUnauthorized
	}
	return segments[0], nil
}

func (s *Services) optionalUserID(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	return s.parseToken(token)
}

func calculatePreliminaryAssessment(vehicle domain.ValuationVehicleInput) domain.ValuationAssessment {
	expectedPrice := parseFormNumber(vehicle.ExpectedPriceTHB)
	mileage := parseFormNumber(vehicle.MileageKM)
	year := parseFormNumber(vehicle.Year)
	age := time.Now().Year() - year
	if age < 0 {
		age = 0
	}

	mileageFactor := 1.0
	if mileage > 160000 {
		mileageFactor = 0.9
	} else if mileage > 90000 {
		mileageFactor = 0.95
	}

	ageFactor := 1 - float64(age)*0.012
	if ageFactor < 0.86 {
		ageFactor = 0.86
	}

	basePrice := expectedPrice
	if basePrice < 300000 {
		basePrice = 300000
	}
	marketPriceTHB := roundToThousand(float64(basePrice) * ageFactor * mileageFactor)
	dealerBuyPriceTHB := roundToThousand(float64(marketPriceTHB) * 0.82)
	recommendedListPriceTHB := roundToThousand(float64(marketPriceTHB) * 0.94)

	return domain.ValuationAssessment{
		MarketPriceTHB:          marketPriceTHB,
		DealerBuyPriceTHB:       dealerBuyPriceTHB,
		RecommendedListPriceTHB: recommendedListPriceTHB,
		Note:                    "Preliminary estimate from submitted vehicle data. Use it as a starting point for negotiation.",
		EstimatedAt:             time.Now().UTC().Format(time.RFC3339),
	}
}

func buildAssessmentMessage(assessment domain.ValuationAssessment) string {
	return strings.Join([]string{
		"Admin assessment completed.",
		fmt.Sprintf("Estimated market price: %s", formatTHB(assessment.MarketPriceTHB)),
		fmt.Sprintf("Dealer buy price: %s", formatTHB(assessment.DealerBuyPriceTHB)),
		fmt.Sprintf("Recommended listing price: %s", formatTHB(assessment.RecommendedListPriceTHB)),
		assessment.Note,
	}, "\n")
}

func formatTHB(value int64) string {
	return fmt.Sprintf("THB %d", value)
}

func parseFormNumber(value string) int {
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

func roundToThousand(value float64) int64 {
	return int64(value/1000+0.5) * 1000
}

func (s *Services) invalidateVehicleCache(ctx context.Context) error {
	keys, err := s.cache.Keys(ctx, "vehicles:*").Result()
	if err != nil || len(keys) == 0 {
		return err
	}
	return s.cache.Del(ctx, keys...).Err()
}
