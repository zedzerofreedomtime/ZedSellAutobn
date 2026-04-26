package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"zedsellauto/internal/domain"
	"zedsellauto/internal/service"
)

type handler struct {
	services *service.Services
}

func newHandler(services *service.Services) *handler {
	return &handler{services: services}
}

type authRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"fullName"`
}

type offerRequest struct {
	VehicleID      string `json:"vehicleId" binding:"required"`
	FullName       string `json:"fullName" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Phone          string `json:"phone" binding:"required"`
	OfferAmountTHB int64  `json:"offerAmountTHB" binding:"required"`
	Note           string `json:"note"`
}

type testDriveRequest struct {
	VehicleID   string `json:"vehicleId" binding:"required"`
	FullName    string `json:"fullName" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Phone       string `json:"phone" binding:"required"`
	PreferredAt string `json:"preferredAt" binding:"required"`
	Note        string `json:"note"`
}

type inquiryRequest struct {
	VehicleID string `json:"vehicleId" binding:"required"`
	FullName  string `json:"fullName" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Channel   string `json:"channel"`
}

type financeRequest struct {
	VehicleID          string  `json:"vehicleId" binding:"required"`
	FullName           string  `json:"fullName" binding:"required"`
	Email              string  `json:"email" binding:"required,email"`
	Phone              string  `json:"phone" binding:"required"`
	DownPaymentPercent float64 `json:"downPaymentPercent" binding:"required"`
	LoanTermMonths     int     `json:"loanTermMonths" binding:"required"`
	CreditBand         string  `json:"creditBand" binding:"required"`
	MonthlyIncomeTHB   int64   `json:"monthlyIncomeTHB" binding:"required"`
}

type sellerVehicleRequest struct {
	Brand         string   `json:"brand" binding:"required"`
	Model         string   `json:"model" binding:"required"`
	Year          int      `json:"year" binding:"required"`
	PriceTHB      int64    `json:"priceTHB" binding:"required"`
	Location      string   `json:"location" binding:"required"`
	MileageKM     int      `json:"mileageKM" binding:"required"`
	Transmission  string   `json:"transmission"`
	FuelType      string   `json:"fuelType"`
	DriveTrain    string   `json:"driveTrain"`
	Engine        string   `json:"engine"`
	ExteriorColor string   `json:"exteriorColor"`
	InteriorColor string   `json:"interiorColor"`
	OwnerSummary  string   `json:"ownerSummary"`
	SellerName    string   `json:"sellerName" binding:"required"`
	Phone         string   `json:"phone" binding:"required"`
	Email         string   `json:"email" binding:"required,email"`
	Description   string   `json:"description"`
	ImageNames    []string `json:"imageNames"`
}

func (h *handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, h.services.Health(c.Request.Context()))
}

func (h *handler) signup(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.services.Signup(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "accessToken": token})
}

func (h *handler) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.services.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "accessToken": token})
}

func (h *handler) me(c *gin.Context) {
	user, err := h.services.Me(c.Request.Context(), bearerToken(c))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *handler) home(c *gin.Context) {
	payload, err := h.services.Home(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *handler) vehicleCategories(c *gin.Context) {
	payload, err := h.services.VehicleCategories(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": payload})
}

func (h *handler) vehicles(c *gin.Context) {
	category := c.Query("category")
	featuredOnly := c.Query("featured") == "true"
	payload, err := h.services.Vehicles(c.Request.Context(), category, featuredOnly)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"vehicles": payload})
}

func (h *handler) vehicleDetail(c *gin.Context) {
	payload, err := h.services.VehicleDetail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *handler) blogPosts(c *gin.Context) {
	payload, err := h.services.BlogPosts(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": payload})
}

func (h *handler) blogDetail(c *gin.Context) {
	payload, err := h.services.BlogDetail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *handler) pricing(c *gin.Context) {
	payload, err := h.services.Pricing(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *handler) howItWorks(c *gin.Context) {
	payload, err := h.services.HowItWorks(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *handler) favorites(c *gin.Context) {
	payload, err := h.services.Favorites(c.Request.Context(), bearerToken(c))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorites": payload})
}

func (h *handler) addFavorite(c *gin.Context) {
	if err := h.services.AddFavorite(c.Request.Context(), bearerToken(c), c.Param("vehicleID")); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *handler) removeFavorite(c *gin.Context) {
	if err := h.services.RemoveFavorite(c.Request.Context(), bearerToken(c), c.Param("vehicleID")); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) createOffer(c *gin.Context) {
	var req offerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.services.CreateOffer(c.Request.Context(), bearerToken(c), req.VehicleID, req.FullName, req.Email, req.Phone, req.OfferAmountTHB, req.Note); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *handler) createTestDrive(c *gin.Context) {
	var req testDriveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	preferredAt, err := time.Parse(time.RFC3339, req.PreferredAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preferredAt must be RFC3339"})
		return
	}
	if err := h.services.CreateTestDrive(c.Request.Context(), bearerToken(c), req.VehicleID, req.FullName, req.Email, req.Phone, preferredAt, req.Note); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *handler) createInquiry(c *gin.Context) {
	var req inquiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.services.CreateInquiry(c.Request.Context(), bearerToken(c), req.VehicleID, req.FullName, req.Email, req.Phone, req.Message, req.Channel); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *handler) createFinance(c *gin.Context) {
	var req financeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.services.CreateFinanceApplication(c.Request.Context(), bearerToken(c), req.VehicleID, req.FullName, req.Email, req.Phone, req.DownPaymentPercent, req.LoanTermMonths, req.CreditBand, req.MonthlyIncomeTHB); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *handler) createSellerVehicle(c *gin.Context) {
	var req sellerVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.services.CreateSellerVehicleSubmission(c.Request.Context(), bearerToken(c), domain.SellerVehicleSubmissionInput{
		Brand:         req.Brand,
		Model:         req.Model,
		Year:          req.Year,
		PriceTHB:      req.PriceTHB,
		Location:      req.Location,
		MileageKM:     req.MileageKM,
		Transmission:  req.Transmission,
		FuelType:      req.FuelType,
		DriveTrain:    req.DriveTrain,
		Engine:        req.Engine,
		ExteriorColor: req.ExteriorColor,
		InteriorColor: req.InteriorColor,
		OwnerSummary:  req.OwnerSummary,
		SellerName:    req.SellerName,
		Phone:         req.Phone,
		Email:         req.Email,
		Description:   req.Description,
		ImageNames:    req.ImageNames,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}
