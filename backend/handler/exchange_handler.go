package handler

import (
	"exchange_cdk/middleware"
	"exchange_cdk/pkg/jwt"
	"exchange_cdk/service"

	"github.com/gin-gonic/gin"
)

type ExchangeHandler struct {
	svc *service.ExchangeService
}

func NewExchangeHandler(svc *service.ExchangeService) *ExchangeHandler {
	return &ExchangeHandler{svc: svc}
}

type exchangeReq struct {
	Amount   float64 `json:"amount" binding:"required"`
	Quantity uint    `json:"quantity" binding:"required"`
}

// Exchange POST /api/exchange
func (h *ExchangeHandler) Exchange(c *gin.Context) {
	var req exchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	claims := c.MustGet(middleware.ClaimsKey).(*jwt.Claims)
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	result, err := h.svc.Exchange(claims.UserID, req.Amount, req.Quantity, ip, ua)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": result})
}

// Amounts GET /api/amounts
func (h *ExchangeHandler) Amounts(c *gin.Context) {
	list, err := h.svc.GetAvailableAmounts()
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": list})
}
