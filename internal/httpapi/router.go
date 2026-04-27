package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"zedsellauto/internal/config"
	"zedsellauto/internal/repository"
	"zedsellauto/internal/service"
)

func NewRouter(cfg config.Config, services *service.Services) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), cors(cfg.CORSAllowedOrigins))

	handler := newHandler(services)

	router.GET("/healthz", handler.health)
	router.GET("/readyz", handler.health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/signup", handler.signup)
		api.POST("/auth/login", handler.login)
		api.GET("/me", handler.me)

		api.GET("/home", handler.home)
		api.GET("/vehicles/categories", handler.vehicleCategories)
		api.GET("/vehicles", handler.vehicles)
		api.GET("/vehicles/:slug", handler.vehicleDetail)

		api.GET("/blog/posts", handler.blogPosts)
		api.GET("/blog/posts/:slug", handler.blogDetail)

		api.GET("/resources/pricing", handler.pricing)
		api.GET("/resources/how-it-works", handler.howItWorks)

		api.GET("/favorites", handler.favorites)
		api.POST("/favorites/:vehicleID", handler.addFavorite)
		api.DELETE("/favorites/:vehicleID", handler.removeFavorite)

		api.POST("/leads/offers", handler.createOffer)
		api.POST("/leads/test-drives", handler.createTestDrive)
		api.POST("/leads/inquiries", handler.createInquiry)
		api.POST("/leads/finance", handler.createFinance)

		api.POST("/seller/vehicles", handler.createSellerVehicle)
		api.GET("/seller/valuations", handler.sellerValuations)
		api.POST("/seller/valuations", handler.createSellerValuation)
		api.POST("/seller/valuations/:id/messages", handler.addSellerValuationMessage)
		api.POST("/seller/valuations/:id/publish", handler.publishSellerValuation)
		api.GET("/seller/listings", handler.sellerListings)
		api.GET("/seller/listings/:id", handler.sellerListingDetail)

		api.POST("/admin/valuations/:id/messages", handler.addAdminValuationMessage)
		api.POST("/admin/valuations/:id/assessment", handler.sendAdminValuationAssessment)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})

	return router
}

func cors(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed["*"]; ok {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return ""
}

func handleError(c *gin.Context, err error) {
	switch {
	case err == nil:
		return
	case err == service.ErrUnauthorized:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case err == service.ErrConflict:
		c.JSON(http.StatusConflict, gin.H{"error": "resource already exists"})
	case err == repository.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
