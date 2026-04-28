package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'buyer',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vehicle_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  image_url TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS vehicles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  category_slug TEXT NOT NULL REFERENCES vehicle_categories(slug),
  name TEXT NOT NULL,
  year INT NOT NULL,
  price_thb BIGINT NOT NULL,
  monthly_payment_thb BIGINT NOT NULL,
  location TEXT NOT NULL,
  mileage_km INT NOT NULL,
  fuel_type TEXT NOT NULL,
  tag TEXT NOT NULL,
  tone TEXT NOT NULL,
  image_url TEXT NOT NULL,
  gallery JSONB NOT NULL DEFAULT '[]'::jsonb,
  transmission TEXT NOT NULL,
  drive_train TEXT NOT NULL,
  engine TEXT NOT NULL,
  exterior_color TEXT NOT NULL,
  interior_color TEXT NOT NULL,
  seats INT NOT NULL,
  owner_summary TEXT NOT NULL,
  description TEXT NOT NULL,
  seller_name TEXT NOT NULL,
  seller_email_verified BOOLEAN NOT NULL DEFAULT TRUE,
  seller_phone_verified BOOLEAN NOT NULL DEFAULT TRUE,
  seller_zed_pay_ready BOOLEAN NOT NULL DEFAULT TRUE,
  is_featured BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pricing_highlights (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  label TEXT NOT NULL UNIQUE,
  value TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pricing_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL,
  price_label TEXT NOT NULL,
  highlight TEXT NOT NULL DEFAULT '',
  features JSONB NOT NULL DEFAULT '[]'::jsonb,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pricing_faqs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  question TEXT NOT NULL UNIQUE,
  answer TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS how_it_works_steps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  label TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS trust_signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL,
  icon TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS experience_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  audience TEXT NOT NULL,
  content TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  UNIQUE (audience, content)
);

CREATE TABLE IF NOT EXISTS blog_posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL,
  title TEXT NOT NULL,
  excerpt TEXT NOT NULL,
  image_url TEXT NOT NULL,
  published_at DATE NOT NULL,
  read_time_minutes INT NOT NULL,
  author TEXT NOT NULL,
  sections JSONB NOT NULL DEFAULT '[]'::jsonb,
  is_featured BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS favorites (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, vehicle_id)
);

CREATE TABLE IF NOT EXISTS lead_offers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id UUID NOT NULL REFERENCES vehicles(id),
  user_id UUID REFERENCES users(id),
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone TEXT NOT NULL,
  offer_amount_thb BIGINT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lead_test_drives (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id UUID NOT NULL REFERENCES vehicles(id),
  user_id UUID REFERENCES users(id),
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone TEXT NOT NULL,
  preferred_at TIMESTAMPTZ NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lead_inquiries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id UUID NOT NULL REFERENCES vehicles(id),
  user_id UUID REFERENCES users(id),
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone TEXT NOT NULL,
  message TEXT NOT NULL,
  channel TEXT NOT NULL DEFAULT 'chat',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS finance_applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id UUID NOT NULL REFERENCES vehicles(id),
  user_id UUID REFERENCES users(id),
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone TEXT NOT NULL,
  down_payment_percent NUMERIC(5,2) NOT NULL,
  loan_term_months INT NOT NULL,
  credit_band TEXT NOT NULL,
  monthly_income_thb BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS seller_vehicle_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  brand TEXT NOT NULL,
  model TEXT NOT NULL,
  year INT NOT NULL,
  price_thb BIGINT NOT NULL,
  location TEXT NOT NULL,
  mileage_km INT NOT NULL,
  transmission TEXT NOT NULL DEFAULT '',
  fuel_type TEXT NOT NULL DEFAULT '',
  drive_train TEXT NOT NULL DEFAULT '',
  engine TEXT NOT NULL DEFAULT '',
  exterior_color TEXT NOT NULL DEFAULT '',
  interior_color TEXT NOT NULL DEFAULT '',
  owner_summary TEXT NOT NULL DEFAULT '',
  seller_name TEXT NOT NULL,
  phone TEXT NOT NULL,
  email TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  image_names JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE seller_vehicle_submissions
  ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS seller_valuation_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  status TEXT NOT NULL DEFAULT 'pending',
  vehicle JSONB NOT NULL,
  contact JSONB NOT NULL,
  preliminary_assessment JSONB NOT NULL,
  final_assessment JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS seller_valuation_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id UUID NOT NULL REFERENCES seller_valuation_requests(id) ON DELETE CASCADE,
  sender TEXT NOT NULL,
  text TEXT NOT NULL,
  assessment JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS seller_listings (
  id TEXT PRIMARY KEY DEFAULT ('seller-listing-' || gen_random_uuid()::text),
  source_request_id UUID REFERENCES seller_valuation_requests(id) ON DELETE SET NULL,
  source_submission_id UUID REFERENCES seller_vehicle_submissions(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'published',
  category_slug TEXT NOT NULL DEFAULT 'sedan' REFERENCES vehicle_categories(slug),
  title TEXT NOT NULL,
  price_thb BIGINT NOT NULL,
  image_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by_email TEXT NOT NULL DEFAULT '',
  vehicle JSONB NOT NULL,
  contact JSONB NOT NULL,
  listed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS seller_listings_source_request_unique
  ON seller_listings(source_request_id)
  WHERE source_request_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS seller_listings_source_submission_unique
  ON seller_listings(source_submission_id)
  WHERE source_submission_id IS NOT NULL;

ALTER TABLE lead_offers
  ALTER COLUMN vehicle_id DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS seller_listing_id TEXT REFERENCES seller_listings(id);

ALTER TABLE lead_test_drives
  ALTER COLUMN vehicle_id DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS seller_listing_id TEXT REFERENCES seller_listings(id);

ALTER TABLE lead_inquiries
  ALTER COLUMN vehicle_id DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS seller_listing_id TEXT REFERENCES seller_listings(id);

ALTER TABLE finance_applications
  ALTER COLUMN vehicle_id DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS seller_listing_id TEXT REFERENCES seller_listings(id);

CREATE TABLE IF NOT EXISTS market_used_car_prices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  source_url TEXT NOT NULL UNIQUE,
  brand TEXT NOT NULL,
  model TEXT NOT NULL,
  raw_model TEXT NOT NULL,
  model_year INT NOT NULL,
  model_month INT NOT NULL DEFAULT 0,
  monthly_payment_thb INT NOT NULL DEFAULT 0,
  price_min_thb BIGINT NOT NULL,
  price_max_thb BIGINT NOT NULL,
  imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS market_used_car_prices_brand_model_year_idx
  ON market_used_car_prices (LOWER(brand), LOWER(model), model_year);
`

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, schemaSQL)
	return err
}
