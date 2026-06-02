package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminRedeemCategoryHandler struct {
	svc *service.RedeemCategoryService
}

func NewAdminRedeemCategoryHandler(svc *service.RedeemCategoryService) *AdminRedeemCategoryHandler {
	return &AdminRedeemCategoryHandler{svc: svc}
}

type redeemCategoryReq struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"`
}

// List GET /api/admin/redeem-categories
func (h *AdminRedeemCategoryHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := h.svc.List(page, pageSize, c.Query("keyword"), c.Query("status"), getUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// Create POST /api/admin/redeem-categories
func (h *AdminRedeemCategoryHandler) Create(c *gin.Context) {
	var req redeemCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	category, err := h.svc.Create(req.Name, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": category})
}

// Update PUT /api/admin/redeem-categories/:id
func (h *AdminRedeemCategoryHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req redeemCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.Update(uint(id), req.Name, req.Status, getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

// Delete DELETE /api/admin/redeem-categories/:id
func (h *AdminRedeemCategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	if err := h.svc.Delete(uint(id), getUserID(c)); err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "删除失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}
