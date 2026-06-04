package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminInventoryHandler struct {
	svc *service.InventoryService
}

func NewAdminInventoryHandler(svc *service.InventoryService) *AdminInventoryHandler {
	return &AdminInventoryHandler{svc: svc}
}

// ListTemplates GET /api/admin/inventory/templates
func (h *AdminInventoryHandler) ListTemplates(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := h.svc.ListTemplateInventory(service.InventoryListInput{
		Page:          page,
		PageSize:      pageSize,
		Keyword:       c.Query("keyword"),
		Status:        c.Query("status"),
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

// UpdatePolicy PUT /api/admin/templates/:id/inventory-policy
func (h *AdminInventoryHandler) UpdatePolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req service.InventoryPolicyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.UpdatePolicy(uint(id), req, getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

// Replenish POST /api/admin/templates/:id/replenish
func (h *AdminInventoryHandler) Replenish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	list, err := h.svc.ReplenishTemplate(uint(id), getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": len(list),
	}})
}
