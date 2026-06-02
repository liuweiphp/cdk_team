package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminPurchaseTaskHandler struct {
	svc *service.PurchaseTaskService
}

func NewAdminPurchaseTaskHandler(svc *service.PurchaseTaskService) *AdminPurchaseTaskHandler {
	return &AdminPurchaseTaskHandler{svc: svc}
}

type manualCompletePurchaseTaskReq struct {
	SubscribeURL string `json:"subscribe_url" binding:"required"`
}

// List GET /api/admin/purchase-tasks
func (h *AdminPurchaseTaskHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := h.svc.List(service.PurchaseTaskListInput{
		Page:          page,
		PageSize:      pageSize,
		Status:        c.Query("status"),
		PaymentStatus: c.Query("payment_status"),
		CurrentUserID: getUserID(c),
	})
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// ManualComplete POST /api/admin/purchase-tasks/:id/manual-complete
func (h *AdminPurchaseTaskHandler) ManualComplete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req manualCompletePurchaseTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	task, err := h.svc.ManualComplete(uint(id), req.SubscribeURL, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": task})
}
