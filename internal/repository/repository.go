package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
			CASE WHEN c.slug = 'all' THEN (SELECT COUNT(*) FROM vehicles)
			     ELSE (SELECT COUNT(*) FROM vehicles v WHERE v.category_slug = c.slug)
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
	return items, rows.Err()
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
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_offers (vehicle_id, user_id, full_name, email, phone, offer_amount_thb, note)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)
	`, vehicleID, userID, fullName, strings.ToLower(email), phone, amount, note)
	return err
}

func (r *Repository) CreateTestDrive(ctx context.Context, vehicleID, userID, fullName, email, phone string, preferredAt time.Time, note string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_test_drives (vehicle_id, user_id, full_name, email, phone, preferred_at, note)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)
	`, vehicleID, userID, fullName, strings.ToLower(email), phone, preferredAt, note)
	return err
}

func (r *Repository) CreateInquiry(ctx context.Context, vehicleID, userID, fullName, email, phone, message, channel string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO lead_inquiries (vehicle_id, user_id, full_name, email, phone, message, channel)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)
	`, vehicleID, userID, fullName, strings.ToLower(email), phone, message, channel)
	return err
}

func (r *Repository) CreateFinanceApplication(ctx context.Context, vehicleID, userID, fullName, email, phone string, downPercent float64, loanTerm int, creditBand string, income int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO finance_applications (vehicle_id, user_id, full_name, email, phone, down_payment_percent, loan_term_months, credit_band, monthly_income_thb)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9)
	`, vehicleID, userID, fullName, strings.ToLower(email), phone, downPercent, loanTerm, creditBand, income)
	return err
}

func (r *Repository) CreateSellerVehicleSubmission(ctx context.Context, userID string, input domain.SellerVehicleSubmissionInput) (string, error) {
	imageNames, err := json.Marshal(input.ImageNames)
	if err != nil {
		return "", err
	}

	var id string
	err = r.db.QueryRow(ctx, `
		INSERT INTO seller_vehicle_submissions (
			user_id, brand, model, year, price_thb, location, mileage_km, transmission, fuel_type,
			drive_train, engine, exterior_color, interior_color, owner_summary, seller_name, phone,
			email, description, image_names
		)
		VALUES (
			NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19
		)
		RETURNING id
	`, userID, input.Brand, input.Model, input.Year, input.PriceTHB, input.Location, input.MileageKM,
		input.Transmission, input.FuelType, input.DriveTrain, input.Engine, input.ExteriorColor,
		input.InteriorColor, input.OwnerSummary, input.SellerName, input.Phone, strings.ToLower(input.Email),
		input.Description, imageNames).Scan(&id)
	return id, err
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
