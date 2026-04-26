package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"fullName"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type VehicleCategory struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Count       int    `json:"count"`
}

type Vehicle struct {
	ID                   string   `json:"id"`
	Slug                 string   `json:"slug"`
	CategorySlug         string   `json:"categorySlug"`
	Name                 string   `json:"name"`
	Year                 int      `json:"year"`
	PriceTHB             int64    `json:"priceTHB"`
	MonthlyPaymentTHB    int64    `json:"monthlyPaymentTHB"`
	Location             string   `json:"location"`
	MileageKM            int      `json:"mileageKM"`
	FuelType             string   `json:"fuelType"`
	Tag                  string   `json:"tag"`
	Tone                 string   `json:"tone"`
	ImageURL             string   `json:"imageUrl"`
	Gallery              []string `json:"gallery"`
	Transmission         string   `json:"transmission"`
	DriveTrain           string   `json:"driveTrain"`
	Engine               string   `json:"engine"`
	ExteriorColor        string   `json:"exteriorColor"`
	InteriorColor        string   `json:"interiorColor"`
	Seats                int      `json:"seats"`
	OwnerSummary         string   `json:"ownerSummary"`
	Description          string   `json:"description"`
	SellerName           string   `json:"sellerName"`
	SellerEmailVerified  bool     `json:"sellerEmailVerified"`
	SellerPhoneVerified  bool     `json:"sellerPhoneVerified"`
	SellerZedPayReady    bool     `json:"sellerZedPayReady"`
	IsFeatured           bool     `json:"isFeatured"`
	EstimatedMarketPrice int64    `json:"estimatedMarketPrice,omitempty"`
	NearbyListingCount   int      `json:"nearbyListingCount,omitempty"`
	AvgDaysOnMarket      int      `json:"avgDaysOnMarket,omitempty"`
}

type PricingHighlight struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type PricingPlan struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PriceLabel  string   `json:"priceLabel"`
	Highlight   string   `json:"highlight,omitempty"`
	Features    []string `json:"features"`
}

type PricingFAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type HowItWorksStep struct {
	Label       string `json:"label"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TrustSignal struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type BlogSection struct {
	Heading string   `json:"heading"`
	Body    []string `json:"body"`
}

type BlogPost struct {
	Slug            string        `json:"slug"`
	Category        string        `json:"category"`
	Title           string        `json:"title"`
	Excerpt         string        `json:"excerpt"`
	ImageURL        string        `json:"imageUrl"`
	PublishedAt     string        `json:"publishedAt"`
	ReadTimeMinutes int           `json:"readTimeMinutes"`
	Author          string        `json:"author"`
	Sections        []BlogSection `json:"sections,omitempty"`
	IsFeatured      bool          `json:"isFeatured"`
}

type HomePayload struct {
	FeaturedVehicles []Vehicle         `json:"featuredVehicles"`
	Categories       []VehicleCategory `json:"categories"`
	FeaturedPost     *BlogPost         `json:"featuredPost,omitempty"`
}

type SellerVehicleSubmissionInput struct {
	Brand         string
	Model         string
	Year          int
	PriceTHB      int64
	Location      string
	MileageKM     int
	Transmission  string
	FuelType      string
	DriveTrain    string
	Engine        string
	ExteriorColor string
	InteriorColor string
	OwnerSummary  string
	SellerName    string
	Phone         string
	Email         string
	Description   string
	ImageNames    []string
}
