package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminCdkHandler struct {
	svc *service.CdkService
}

func NewAdminCdkHandler(svc *service.CdkService) *AdminCdkHandler {
	return &AdminCdkHandler{svc: svc}
}

// List GET /api/admin/cdk/list
func (h *AdminCdkHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	amount := queryFloat(c, "amount", 0)
	status := c.Query("status")
	code := c.Query("code")
	importID := uint(queryInt(c, "import_id", 0))
	itemID := uint(queryInt(c, "item_id", 0))

	list, total, err := h.svc.List(page, pageSize, amount, status, code, importID, itemID, getUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// Delete DELETE /api/admin/cdk/:id
func (h *AdminCdkHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	if err := h.svc.Delete(uint(id), getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}
