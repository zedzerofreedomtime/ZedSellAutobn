package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zedsellauto/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, fullName string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, full_name, role, created_at
	`, strings.ToLower(email), passwordHash, fullName).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
	)
	return user, err
}

func (r *Repository) FindUserAuthByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id, email, full_name, role, created_at, password_hash
		FROM users
		WHERE email = $1
	`, strings.ToLower(email)).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&passwordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", ErrNotFound
	}
	return user, passwordHash, err
}

func (r *Repository) FindUserByID(ctx context.Context, userID string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, full_name, role, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (r *Repository) ListVehicleCategories(ctx context.Context) ([]domain.VehicleCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.slug, c.title, c.description, c.image_url,
			CASE WHEN c.slug = 'all' THEN
				(SELECT COUNT(*) FROM vehicles) +
				(SELECT COUNT(*) FROM seller_listings sl WHERE sl.status = 'published')
			ELSE
				(SELECT COUNT(*) FROM vehicles v WHERE v.category_slug = c.slug) +
				(SELECT COUNT(*) FROM seller_listings sl WHERE sl.category_slug = c.slug AND sl.status = 'published')
			END AS count
		FROM vehicle_categories c
		ORDER BY c.sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.VehicleCategory
	for rows.Next() {
		var item domain.VehicleCategory
		if err := rows.Scan(&item.Slug, &item.Title, &item.Description, &item.ImageURL, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListVehicles(ctx context.Context, category string, featuredOnly bool, limit int) ([]domain.Vehicle, error) {
	query := `
		SELECT id, slug, category_slug, name, year, price_thb, monthly_payment_thb, location, mileage_km, fuel_type, tag, tone,
		       image_url, gallery, transmission, drive_train, engine, exterior_color, interior_color, seats, owner_summary,
		       description, seller_name, seller_email_verified, seller_phone_verified, seller_zed_pay_ready, is_featured
		FROM vehicles
		WHERE ($1 = '' OR category_slug = $1)
		  AND ($2 = FALSE OR is_featured = TRUE)
		ORDER BY is_featured DESC, year DESC, price_thb DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(ctx, query, category, featuredOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Vehicle
	for rows.Next() {
		item, err := scanVehicle(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if featuredOnly {
		return items, nil
	}

	sellerItems, err := r.ListSellerListingVehicles(ctx, category, 0)
	if err != nil {
		return nil, err
	}
	return append(sellerItems, items...), nil
}

func (r *Repository) GetVehicleBySlug(ctx context.Context, slug string) (domain.Vehicle, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, slug, category_slug, name, year, price_thb, monthly_payment_thb, location, mileage_km, fuel_type, tag, tone,
		       image_url, gallery, transmission, drive_train, engine, exterior_color, interior_color, seats, owner_summary,
		       description, seller_name, seller_email_verified, seller_phone_verified, seller_zed_pay_ready, is_featured
		FROM vehicles
		WHERE slug = $1
	`, slug)

	item, err := scanVehicle(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if strings.HasPrefix(slug, "seller-listing-") {
			return r.GetSellerListingVehicleByID(ctx, slug)
		}
		return domain.Vehicle{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListRelatedVehicles(ctx context.Context, slug, category string, limit int) ([]domain.Vehicle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, slug, category_slug, name, year, price_thb, monthly_payment_thb, location, mileage_km, fuel_type, tag, tone,
		       image_url, gallery, transmission, drive_train, engine, exterior_color, interior_color, seats, owner_summary,
		       description, seller_name, seller_email_verified, seller_phone_verified, seller_zed_pay_ready, is_featured
		FROM vehicles
		WHERE category_slug = $1 AND slug <> $2
		ORDER BY year DESC, price_thb DESC
		LIMIT $3
	`, category, slug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Vehicle
	for rows.Next() {
		item, err := scanVehicle(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListPricingHighlights(ctx context.Context) ([]domain.PricingHighlight, error) {
	rows, err := r.db.Query(ctx, `SELECT label, value FROM pricing_highlights ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PricingHighlight
	for rows.Next() {
		var item domain.PricingHighlight
		if err := rows.Scan(&item.Label, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListPricingPlans(ctx context.Context) ([]domain.PricingPlan, error) {
	rows, err := r.db.Query(ctx, `SELECT title, description, price_label, highlight, features FROM pricing_plans ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PricingPlan
	for rows.Next() {
		var item domain.PricingPlan
		var featuresRaw []byte
		if err := rows.Scan(&item.Title, &item.Description, &item.PriceLabel, &item.Highlight, &featuresRaw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(featuresRaw, &item.Features); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListPricingFAQs(ctx context.Context) ([]domain.PricingFAQ, error) {
	rows, err := r.db.Query(ctx, `SELECT question, answer FROM pricing_faqs ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.PricingFAQ
	for rows.Next() {
		var item domain.PricingFAQ
		if err := rows.Scan(&item.Question, &item.Answer); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListHowItWorksSteps(ctx context.Context) ([]domain.HowItWorksStep, error) {
	rows, err := r.db.Query(ctx, `SELECT label, title, description FROM how_it_works_steps ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.HowItWorksStep
	for rows.Next() {
		var item domain.HowItWorksStep
		if err := rows.Scan(&item.Label, &item.Title, &item.Description); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListTrustSignals(ctx context.Context) ([]domain.TrustSignal, error) {
	rows, err := r.db.Query(ctx, `SELECT title, description, icon FROM trust_signals ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TrustSignal
	for rows.Next() {
		var item domain.TrustSignal
		if err := rows.Scan(&item.Title, &item.Description, &item.Icon); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListExperienceItems(ctx context.Context, audience string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT content FROM experience_items WHERE audience = $1 ORDER BY sort_order`, audience)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListBlogPosts(ctx context.Context) ([]domain.BlogPost, error) {
	rows, err := r.db.Query(ctx, `
		SELECT slug, category, title, excerpt, image_url, published_at, read_time_minutes, author, is_featured
		FROM blog_posts
		ORDER BY is_featured DESC, published_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.BlogPost
	for rows.Next() {
		var item domain.BlogPost
		var publishedAt time.Time
		if err := rows.Scan(&item.Slug, &item.Category, &item.Title, &item.Excerpt, &item.ImageURL, &publishedAt, &item.ReadTimeMinutes, &item.Author, &item.IsFeatured); err != nil {
			return nil, err
		}
		item.PublishedAt = publishedAt.Format("2006-01-02")
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetBlogPostBySlug(ctx context.Context, slug string) (domain.BlogPost, error) {
	var item domain.BlogPost
	var sectionsRaw []byte
	var publishedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT slug, category, title, excerpt, image_url, published_at, read_time_minutes, author, sections, is_featured
		FROM blog_posts
		WHERE slug = $1
	`, slug).Scan(
		&item.Slug, &item.Category, &item.Title, &item.Excerpt, &item.ImageURL, &publishedAt,
		&item.ReadTimeMinutes, &item.Author, &sectionsRaw, &item.IsFeatured,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BlogPost{}, ErrNotFound
	}
	if err != nil {
		return domain.BlogPost{}, err
	}
	item.PublishedAt = publishedAt.Format("2006-01-02")
	if err := json.Unmarshal(sectionsRaw, &item.Sections); err != nil {
		return domain.BlogPost{}, err
	}
	return item, nil
}

func (r *Repository) ListRelatedBlogPosts(ctx context.Context, slug, category string, limit int) ([]domain.BlogPost, error) {
	rows, err := r.db.Query(ctx, `
		SELECT slug, category, title, excerpt, image_url, published_at, read_time_minutes, author, is_featured
		FROM blog_posts
		WHERE slug <> $1
		ORDER BY CASE WHEN category = $2 THEN 0 ELSE 1 END, published_at DESC
		LIMIT $3
	`, slug, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.BlogPost
	for rows.Next() {
		var item domain.BlogPost
		var publishedAt time.Time
		if err := rows.Scan(&item.Slug, &item.Category, &item.Title, &item.Excerpt, &item.ImageURL, &publishedAt, &item.ReadTimeMinutes, &item.Author, &item.IsFeatured); err != nil {
			return nil, err
		}
		item.PublishedAt = publishedAt.Format("2006-01-02")
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListFavorites(ctx context.Context, userID string) ([]domain.Vehicle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.id, v.slug, v.category_slug, v.name, v.year, v.price_thb, v.monthly_payment_thb, v.location, v.mileage_km, v.fuel_type, v.tag, v.tone,
		       v.image_url, v.gallery, v.transmission, v.drive_train, v.engine, v.exterior_color, v.interior_color, v.seats, v.owner_summary,
		       v.description, v.seller_name, v.seller_email_verified, v.seller_phone_verified, v.seller_zed_pay_ready, v.is_featured
		FROM favorites f
		JOIN vehicles v ON v.id = f.vehicle_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Vehicle
	for rows.Next() {
		item, err := scanVehicle(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AddFavorite(ctx context.Context, userID, vehicleID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO favorites (user_id, vehicle_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, vehicle_id) DO NOTHING
	`, userID, vehicleID)
	return err
}

func (r *Repository) RemoveFavorite(ctx context.Context, userID, vehicleID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1 AND vehicle_id = $2`, userID, vehicleID)
	return err
}

func (r *Repository) CreateOffer(ctx context.Context, vehicleID, userID, fullName, email, phone string, amount int64, note string) error {
	vehicleRef, listingRef := splitVehicleReference(vehicleID)
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_offers (vehicle_id, seller_listing_id, user_id, full_name, email, phone, offer_amount_thb, note)
		VALUES (NULLIF($1, '')::uuid, NULLIF($2, ''), NULLIF($3, '')::uuid, $4, $5, $6, $7, $8)
	`, vehicleRef, listingRef, userID, fullName, strings.ToLower(email), phone, amount, note)
	return err
}

func (r *Repository) CreateTestDrive(ctx context.Context, vehicleID, userID, fullName, email, phone string, preferredAt time.Time, note string) error {
	vehicleRef, listingRef := splitVehicleReference(vehicleID)
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_test_drives (vehicle_id, seller_listing_id, user_id, full_name, email, phone, preferred_at, note)
		VALUES (NULLIF($1, '')::uuid, NULLIF($2, ''), NULLIF($3, '')::uuid, $4, $5, $6, $7, $8)
	`, vehicleRef, listingRef, userID, fullName, strings.ToLower(email), phone, preferredAt, note)
	return err
}

func (r *Repository) CreateInquiry(ctx context.Context, vehicleID, userID, fullName, email, phone, message, channel string) error {
	vehicleRef, listingRef := splitVehicleReference(vehicleID)
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_inquiries (vehicle_id, seller_listing_id, user_id, full_name, email, phone, message, channel)
		VALUES (NULLIF($1, '')::uuid, NULLIF($2, ''), NULLIF($3, '')::uuid, $4, $5, $6, $7, $8)
	`, vehicleRef, listingRef, userID, fullName, strings.ToLower(email), phone, message, channel)
	return err
}

func (r *Repository) CreateFinanceApplication(ctx context.Context, vehicleID, userID, fullName, email, phone string, downPercent float64, loanTerm int, creditBand string, income int64) error {
	vehicleRef, listingRef := splitVehicleReference(vehicleID)
	_, err := r.db.Exec(ctx, `
		INSERT INTO finance_applications (vehicle_id, seller_listing_id, user_id, full_name, email, phone, down_payment_percent, loan_term_months, credit_band, monthly_income_thb)
		VALUES (NULLIF($1, '')::uuid, NULLIF($2, ''), NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10)
	`, vehicleRef, listingRef, userID, fullName, strings.ToLower(email), phone, downPercent, loanTerm, creditBand, income)
	return err
}

func (r *Repository) CreateSellerVehicleSubmission(ctx context.Context, userID string, input domain.SellerVehicleSubmissionInput) (domain.SellerVehicleSubmissionResult, error) {
	imageNames, err := json.Marshal(input.ImageNames)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}
	imageURLs, err := json.Marshal(input.ImageURLs)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO seller_vehicle_submissions (
			user_id, brand, model, year, price_thb, location, mileage_km, transmission, fuel_type,
			drive_train, engine, exterior_color, interior_color, owner_summary, seller_name, phone,
			email, description, image_names, image_urls, status
		)
		VALUES (
			NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, 'published'
		)
		RETURNING id
	`, userID, input.Brand, input.Model, input.Year, input.PriceTHB, input.Location, input.MileageKM,
		input.Transmission, input.FuelType, input.DriveTrain, input.Engine, input.ExteriorColor,
		input.InteriorColor, input.OwnerSummary, input.SellerName, input.Phone, strings.ToLower(input.Email),
		input.Description, imageNames, imageURLs).Scan(&id)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}

	listing := domain.SellerListing{
		PriceTHB:       input.PriceTHB,
		Status:         "published",
		Title:          buildVehicleTitleFromParts(strconv.Itoa(input.Year), input.Brand, input.Model),
		CategorySlug:   inferSellerListingCategory(input.Brand, input.Model, input.FuelType, input.DriveTrain, input.Engine),
		CreatedByEmail: strings.ToLower(input.Email),
		ImageURLs:      input.ImageURLs,
		Contact: domain.ValuationContactInput{
			SellerName: input.SellerName,
			Phone:      input.Phone,
			Email:      input.Email,
		},
		Vehicle: domain.ValuationVehicleInput{
			Brand:            input.Brand,
			Model:            input.Model,
			Year:             strconv.Itoa(input.Year),
			ExpectedPriceTHB: strconv.FormatInt(input.PriceTHB, 10),
			Location:         input.Location,
			MileageKM:        strconv.Itoa(input.MileageKM),
			Transmission:     input.Transmission,
			FuelType:         input.FuelType,
			DriveTrain:       input.DriveTrain,
			Engine:           input.Engine,
			ExteriorColor:    input.ExteriorColor,
			InteriorColor:    input.InteriorColor,
			OwnerSummary:     input.OwnerSummary,
			ConditionSummary: input.Description,
			Description:      input.Description,
		},
	}

	createdListing, err := createSellerListing(ctx, tx, "", id, listing)
	if err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SellerVehicleSubmissionResult{}, err
	}

	return domain.SellerVehicleSubmissionResult{
		ID:        id,
		ListingID: createdListing.ID,
		Status:    "published",
	}, nil
}

func (r *Repository) CreateValuationRequest(ctx context.Context, userID string, input domain.CreateValuationInput, preliminary domain.ValuationAssessment) (domain.ValuationRequest, error) {
	vehicle, err := json.Marshal(input.Vehicle)
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	contact, err := json.Marshal(input.Contact)
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	assessment, err := json.Marshal(preliminary)
	if err != nil {
		return domain.ValuationRequest{}, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	defer tx.Rollback(ctx)

	var requestID string
	err = tx.QueryRow(ctx, `
		INSERT INTO seller_valuation_requests (user_id, vehicle, contact, preliminary_assessment)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4)
		RETURNING id
	`, userID, vehicle, contact, assessment).Scan(&requestID)
	if err != nil {
		return domain.ValuationRequest{}, err
	}

	title := buildVehicleTitle(input.Vehicle)
	if title == "" {
		title = "seller vehicle"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO seller_valuation_messages (request_id, sender, text)
		VALUES ($1, 'seller', $2), ($1, 'admin', $3)
	`, requestID,
		fmt.Sprintf("Submitted %s for preliminary valuation.", title),
		"Your valuation request is saved. You can publish it or chat with admin for a final assessment.",
	); err != nil {
		return domain.ValuationRequest{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ValuationRequest{}, err
	}
	return r.GetValuationRequest(ctx, requestID)
}

func (r *Repository) ListValuationRequests(ctx context.Context) ([]domain.ValuationRequest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, status, vehicle, contact, preliminary_assessment, final_assessment, created_at, updated_at
		FROM seller_valuation_requests
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ValuationRequest
	for rows.Next() {
		item, err := scanValuationRequest(rows)
		if err != nil {
			return nil, err
		}
		if err := r.attachValuationChildren(ctx, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetValuationRequest(ctx context.Context, requestID string) (domain.ValuationRequest, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, status, vehicle, contact, preliminary_assessment, final_assessment, created_at, updated_at
		FROM seller_valuation_requests
		WHERE id = $1
	`, requestID)
	item, err := scanValuationRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ValuationRequest{}, ErrNotFound
	}
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	if err := r.attachValuationChildren(ctx, &item); err != nil {
		return domain.ValuationRequest{}, err
	}
	return item, nil
}

func (r *Repository) AddValuationMessage(ctx context.Context, requestID, sender, text string, assessment *domain.ValuationAssessment) (domain.ValuationMessage, error) {
	var assessmentRaw []byte
	if assessment != nil {
		encoded, err := json.Marshal(assessment)
		if err != nil {
			return domain.ValuationMessage{}, err
		}
		assessmentRaw = encoded
	}

	var message domain.ValuationMessage
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO seller_valuation_messages (request_id, sender, text, assessment)
		VALUES ($1, $2, $3, NULLIF($4, '')::jsonb)
		RETURNING id, sender, text, created_at
	`, requestID, sender, text, string(assessmentRaw)).Scan(&message.ID, &message.Sender, &message.Text, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ValuationMessage{}, ErrNotFound
	}
	if err != nil {
		return domain.ValuationMessage{}, err
	}
	message.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	message.Assessment = assessment

	_, err = r.db.Exec(ctx, `UPDATE seller_valuation_requests SET updated_at = NOW() WHERE id = $1`, requestID)
	return message, err
}

func (r *Repository) SetValuationAssessment(ctx context.Context, requestID string, assessment domain.ValuationAssessment) error {
	encoded, err := json.Marshal(assessment)
	if err != nil {
		return err
	}
	commandTag, err := r.db.Exec(ctx, `
		UPDATE seller_valuation_requests
		SET status = 'assessed', final_assessment = $2, updated_at = NOW()
		WHERE id = $1
	`, requestID, encoded)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateSellerListingForValuation(ctx context.Context, requestID string, askingPriceTHB int64) (domain.SellerListing, error) {
	if existing, err := r.GetSellerListingBySourceRequest(ctx, requestID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return domain.SellerListing{}, err
	}

	request, err := r.GetValuationRequest(ctx, requestID)
	if err != nil {
		return domain.SellerListing{}, err
	}

	price := askingPriceTHB
	if price <= 0 {
		assessment := request.PreliminaryAssessment
		if request.FinalAssessment != nil {
			assessment = *request.FinalAssessment
		}
		price = assessment.RecommendedListPriceTHB
	}

	listing := domain.SellerListing{
		SourceRequestID: request.ID,
		Status:          "published",
		Title:           buildVehicleTitle(request.Vehicle),
		PriceTHB:        price,
		CategorySlug:    inferSellerListingCategory(request.Vehicle.Brand, request.Vehicle.Model, request.Vehicle.FuelType, request.Vehicle.DriveTrain, request.Vehicle.Engine),
		CreatedByEmail:  strings.ToLower(request.Contact.Email),
		Vehicle:         request.Vehicle,
		Contact:         request.Contact,
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.SellerListing{}, err
	}
	defer tx.Rollback(ctx)

	created, err := createSellerListing(ctx, tx, request.ID, "", listing)
	if err != nil {
		return domain.SellerListing{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE seller_valuation_requests
		SET status = 'assessed', updated_at = NOW()
		WHERE id = $1
	`, requestID); err != nil {
		return domain.SellerListing{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SellerListing{}, err
	}
	return created, nil
}

func (r *Repository) ListSellerListings(ctx context.Context, category string) ([]domain.SellerListing, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(source_request_id::text, ''), COALESCE(source_submission_id::text, ''), status, category_slug,
		       title, price_thb, image_urls, created_by_email, vehicle, contact, listed_at
		FROM seller_listings
		WHERE status = 'published'
		  AND ($1 = '' OR category_slug = $1)
		ORDER BY listed_at DESC
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.SellerListing
	for rows.Next() {
		item, err := scanSellerListing(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetSellerListingByID(ctx context.Context, listingID string) (domain.SellerListing, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(source_request_id::text, ''), COALESCE(source_submission_id::text, ''), status, category_slug,
		       title, price_thb, image_urls, created_by_email, vehicle, contact, listed_at
		FROM seller_listings
		WHERE id = $1 AND status = 'published'
	`, listingID)
	item, err := scanSellerListing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SellerListing{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) GetSellerListingBySourceRequest(ctx context.Context, requestID string) (domain.SellerListing, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(source_request_id::text, ''), COALESCE(source_submission_id::text, ''), status, category_slug,
		       title, price_thb, image_urls, created_by_email, vehicle, contact, listed_at
		FROM seller_listings
		WHERE source_request_id = $1 AND status = 'published'
	`, requestID)
	item, err := scanSellerListing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SellerListing{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListSellerListingVehicles(ctx context.Context, category string, limit int) ([]domain.Vehicle, error) {
	listings, err := r.ListSellerListings(ctx, category)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(listings) > limit {
		listings = listings[:limit]
	}

	items := make([]domain.Vehicle, 0, len(listings))
	for _, listing := range listings {
		items = append(items, sellerListingToVehicle(listing))
	}
	return items, nil
}

func (r *Repository) GetSellerListingVehicleByID(ctx context.Context, listingID string) (domain.Vehicle, error) {
	listing, err := r.GetSellerListingByID(ctx, listingID)
	if err != nil {
		return domain.Vehicle{}, err
	}
	return sellerListingToVehicle(listing), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanVehicle(row scanner) (domain.Vehicle, error) {
	var item domain.Vehicle
	var galleryRaw []byte
	err := row.Scan(
		&item.ID, &item.Slug, &item.CategorySlug, &item.Name, &item.Year, &item.PriceTHB, &item.MonthlyPaymentTHB,
		&item.Location, &item.MileageKM, &item.FuelType, &item.Tag, &item.Tone, &item.ImageURL, &galleryRaw,
		&item.Transmission, &item.DriveTrain, &item.Engine, &item.ExteriorColor, &item.InteriorColor,
		&item.Seats, &item.OwnerSummary, &item.Description, &item.SellerName,
		&item.SellerEmailVerified, &item.SellerPhoneVerified, &item.SellerZedPayReady, &item.IsFeatured,
	)
	if err != nil {
		return domain.Vehicle{}, err
	}
	if err := json.Unmarshal(galleryRaw, &item.Gallery); err != nil {
		return domain.Vehicle{}, err
	}
	return item, nil
}

func createSellerListing(ctx context.Context, tx pgx.Tx, sourceRequestID, sourceSubmissionID string, listing domain.SellerListing) (domain.SellerListing, error) {
	vehicle, err := json.Marshal(listing.Vehicle)
	if err != nil {
		return domain.SellerListing{}, err
	}
	contact, err := json.Marshal(listing.Contact)
	if err != nil {
		return domain.SellerListing{}, err
	}
	imageURLs, err := json.Marshal(listing.ImageURLs)
	if err != nil {
		return domain.SellerListing{}, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO seller_listings (
			source_request_id, source_submission_id, status, category_slug, title, price_thb,
			image_urls, created_by_email, vehicle, contact
		)
		VALUES (
			NULLIF($1, '')::uuid, NULLIF($2, '')::uuid, 'published', $3, $4, $5,
			$6, $7, $8, $9
		)
		RETURNING id, COALESCE(source_request_id::text, ''), COALESCE(source_submission_id::text, ''), status,
		          category_slug, title, price_thb, image_urls, created_by_email, vehicle, contact, listed_at
	`, sourceRequestID, sourceSubmissionID, listing.CategorySlug, listing.Title, listing.PriceTHB,
		imageURLs, listing.CreatedByEmail, vehicle, contact)
	return scanSellerListing(row)
}

func scanValuationRequest(row scanner) (domain.ValuationRequest, error) {
	var item domain.ValuationRequest
	var vehicleRaw []byte
	var contactRaw []byte
	var preliminaryRaw []byte
	var finalRaw []byte
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(
		&item.ID,
		&item.Status,
		&vehicleRaw,
		&contactRaw,
		&preliminaryRaw,
		&finalRaw,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.ValuationRequest{}, err
	}
	if err := json.Unmarshal(vehicleRaw, &item.Vehicle); err != nil {
		return domain.ValuationRequest{}, err
	}
	if err := json.Unmarshal(contactRaw, &item.Contact); err != nil {
		return domain.ValuationRequest{}, err
	}
	if err := json.Unmarshal(preliminaryRaw, &item.PreliminaryAssessment); err != nil {
		return domain.ValuationRequest{}, err
	}
	if len(finalRaw) > 0 {
		var assessment domain.ValuationAssessment
		if err := json.Unmarshal(finalRaw, &assessment); err != nil {
			return domain.ValuationRequest{}, err
		}
		item.FinalAssessment = &assessment
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	item.Messages = []domain.ValuationMessage{}
	return item, nil
}

func (r *Repository) attachValuationChildren(ctx context.Context, item *domain.ValuationRequest) error {
	messages, err := r.listValuationMessages(ctx, item.ID)
	if err != nil {
		return err
	}
	item.Messages = messages

	listing, err := r.GetSellerListingBySourceRequest(ctx, item.ID)
	if err == nil {
		item.Listing = &domain.ValuationListing{
			ID:              listing.ID,
			ListedAt:        listing.ListedAt,
			PriceTHB:        listing.PriceTHB,
			SourceRequestID: item.ID,
			Status:          listing.Status,
			Title:           listing.Title,
		}
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (r *Repository) listValuationMessages(ctx context.Context, requestID string) ([]domain.ValuationMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender, text, assessment, created_at
		FROM seller_valuation_messages
		WHERE request_id = $1
		ORDER BY created_at ASC
	`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ValuationMessage
	for rows.Next() {
		var item domain.ValuationMessage
		var assessmentRaw []byte
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Sender, &item.Text, &assessmentRaw, &createdAt); err != nil {
			return nil, err
		}
		if len(assessmentRaw) > 0 {
			var assessment domain.ValuationAssessment
			if err := json.Unmarshal(assessmentRaw, &assessment); err != nil {
				return nil, err
			}
			item.Assessment = &assessment
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSellerListing(row scanner) (domain.SellerListing, error) {
	var item domain.SellerListing
	var sourceSubmissionID string
	var imageURLsRaw []byte
	var vehicleRaw []byte
	var contactRaw []byte
	var listedAt time.Time

	err := row.Scan(
		&item.ID,
		&item.SourceRequestID,
		&sourceSubmissionID,
		&item.Status,
		&item.CategorySlug,
		&item.Title,
		&item.PriceTHB,
		&imageURLsRaw,
		&item.CreatedByEmail,
		&vehicleRaw,
		&contactRaw,
		&listedAt,
	)
	if err != nil {
		return domain.SellerListing{}, err
	}
	if item.SourceRequestID == "" && sourceSubmissionID != "" {
		item.SourceRequestID = "direct-sell-" + sourceSubmissionID
	}
	if err := json.Unmarshal(imageURLsRaw, &item.ImageURLs); err != nil {
		return domain.SellerListing{}, err
	}
	if err := json.Unmarshal(vehicleRaw, &item.Vehicle); err != nil {
		return domain.SellerListing{}, err
	}
	if err := json.Unmarshal(contactRaw, &item.Contact); err != nil {
		return domain.SellerListing{}, err
	}
	item.ListedAt = listedAt.UTC().Format(time.RFC3339)
	return item, nil
}

func sellerListingToVehicle(listing domain.SellerListing) domain.Vehicle {
	year := parseListingNumber(listing.Vehicle.Year)
	if year == 0 {
		year = time.Now().Year()
	}
	mileage := parseListingNumber(listing.Vehicle.MileageKM)
	price := listing.PriceTHB
	if price == 0 {
		price = int64(parseListingNumber(listing.Vehicle.ExpectedPriceTHB))
	}

	imageURL := "/placeholder.svg"
	if len(listing.ImageURLs) > 0 && listing.ImageURLs[0] != "" {
		imageURL = listing.ImageURLs[0]
	}
	gallery := listing.ImageURLs
	if len(gallery) == 0 {
		gallery = []string{imageURL}
	}

	description := listing.Vehicle.Description
	if description == "" {
		description = "Seller listing from Zed Auto"
	}

	return domain.Vehicle{
		ID:                   listing.ID,
		Slug:                 listing.ID,
		CategorySlug:         listing.CategorySlug,
		Name:                 listing.Title,
		Year:                 year,
		PriceTHB:             price,
		MonthlyPaymentTHB:    price / 60,
		Location:             defaultString(listing.Vehicle.Location, "Not specified"),
		MileageKM:            mileage,
		FuelType:             defaultString(listing.Vehicle.FuelType, "Not specified"),
		Tag:                  "Seller listing",
		Tone:                 "success",
		ImageURL:             imageURL,
		Gallery:              gallery,
		Transmission:         defaultString(listing.Vehicle.Transmission, "Not specified"),
		DriveTrain:           defaultString(listing.Vehicle.DriveTrain, "Not specified"),
		Engine:               defaultString(listing.Vehicle.Engine, "Not specified"),
		ExteriorColor:        defaultString(listing.Vehicle.ExteriorColor, "Not specified"),
		InteriorColor:        defaultString(listing.Vehicle.InteriorColor, "Not specified"),
		Seats:                5,
		OwnerSummary:         defaultString(listing.Vehicle.OwnerSummary, "Not specified"),
		Description:          description,
		SellerName:           defaultString(listing.Contact.SellerName, "Zed Auto seller"),
		SellerEmailVerified:  true,
		SellerPhoneVerified:  true,
		SellerZedPayReady:    false,
		IsFeatured:           false,
		EstimatedMarketPrice: int64(float64(price) * 1.04),
		NearbyListingCount:   0,
		AvgDaysOnMarket:      0,
	}
}

func splitVehicleReference(vehicleID string) (string, string) {
	if strings.HasPrefix(vehicleID, "seller-listing-") {
		return "", vehicleID
	}
	return vehicleID, ""
}

func buildVehicleTitle(vehicle domain.ValuationVehicleInput) string {
	return buildVehicleTitleFromParts(vehicle.Year, vehicle.Brand, vehicle.Model)
}

func buildVehicleTitleFromParts(year, brand, model string) string {
	return strings.Join(nonEmptyStrings(year, brand, model), " ")
}

func nonEmptyStrings(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func inferSellerListingCategory(parts ...string) string {
	text := strings.ToLower(strings.Join(parts, " "))
	switch {
	case strings.Contains(text, "ev") || strings.Contains(text, "electric") || strings.Contains(text, "tesla"):
		return "ev"
	case strings.Contains(text, "pickup") || strings.Contains(text, "hilux") || strings.Contains(text, "revo") ||
		strings.Contains(text, "ranger") || strings.Contains(text, "d-max") || strings.Contains(text, "triton"):
		return "pickup"
	case strings.Contains(text, "suv") || strings.Contains(text, "macan") || strings.Contains(text, "q5") ||
		strings.Contains(text, "rx") || strings.Contains(text, "x3") || strings.Contains(text, "x5") ||
		strings.Contains(text, "fortuner") || strings.Contains(text, "pajero") || strings.Contains(text, "cr-v") ||
		strings.Contains(text, "cx-5"):
		return "suv"
	case strings.Contains(text, "porsche") || strings.Contains(text, "audi") || strings.Contains(text, "lexus") ||
		strings.Contains(text, "mercedes") || strings.Contains(text, "benz") || strings.Contains(text, "bmw") ||
		strings.Contains(text, "luxury"):
		return "luxury"
	default:
		return "sedan"
	}
}

func parseListingNumber(value string) int {
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

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
