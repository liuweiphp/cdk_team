package handler

import (
	"exchange_cdk/service"

	"github.com/gin-gonic/gin"
)

type RedeemHandler struct {
	svc *service.RedeemService
}

func NewRedeemHandler(svc *service.RedeemService) *RedeemHandler {
	return &RedeemHandler{svc: svc}
}

type redeemReq struct {
	Code string `json:"code" binding:"required"`
}

// Redeem POST /api/redeem
func (h *RedeemHandler) Redeem(c *gin.Context) {
	var req redeemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "请输入兑换码", "data": nil})
		return
	}
	result, err := h.svc.RedeemByCode(req.Code, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": result})
}
