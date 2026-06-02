package handler

import (
	"exchange_cdk/middleware"
	"exchange_cdk/pkg/jwt"
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc  *service.UserService
	orderSvc *service.ExchangeService
}

func NewUserHandler(userSvc *service.UserService, orderSvc *service.ExchangeService) *UserHandler {
	return &UserHandler{userSvc: userSvc, orderSvc: orderSvc}
}

// Me GET /api/user/me
func (h *UserHandler) Me(c *gin.Context) {
	claims := c.MustGet(middleware.ClaimsKey).(*jwt.Claims)
	user, err := h.userSvc.GetByID(claims.UserID)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "获取用户信息失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": user})
}

// Orders GET /api/user/orders
func (h *UserHandler) Orders(c *gin.Context) {
	claims := c.MustGet(middleware.ClaimsKey).(*jwt.Claims)
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	amount := queryFloat(c, "amount", 0)
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := h.orderSvc.GetUserOrders(claims.UserID, page, pageSize, amount)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// OrderDetail GET /api/user/orders/:id
func (h *UserHandler) OrderDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	order, err := h.orderSvc.GetOrderDetail(uint(id))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "订单不存在", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": order})
}

type changePwdReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword PUT /api/user/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	claims := c.MustGet(middleware.ClaimsKey).(*jwt.Claims)
	var req changePwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.userSvc.ChangePassword(claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "密码修改成功", "data": nil})
}

