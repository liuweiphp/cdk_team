package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminTemplateHandler struct {
	svc *service.TemplateService
}

func NewAdminTemplateHandler(svc *service.TemplateService) *AdminTemplateHandler {
	return &AdminTemplateHandler{svc: svc}
}

type templateReq struct {
	Name               string `json:"name" binding:"required"`
	Content            string `json:"content" binding:"required"`
	Status             string `json:"status"`
	ExternalTargetCode string `json:"external_target_code"`
	ExternalTargetName string `json:"external_target_name"`
}

// List GET /api/admin/templates
func (h *AdminTemplateHandler) List(c *gin.Context) {
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

// Create POST /api/admin/templates
func (h *AdminTemplateHandler) Create(c *gin.Context) {
	var req templateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	tpl, err := h.svc.CreateWithExternal(req.Name, req.Content, req.ExternalTargetCode, req.ExternalTargetName, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": tpl})
}

// Update PUT /api/admin/templates/:id
func (h *AdminTemplateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req templateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.UpdateWithExternal(uint(id), req.Name, req.Content, req.Status, req.ExternalTargetCode, req.ExternalTargetName, getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

// Delete DELETE /api/admin/templates/:id
func (h *AdminTemplateHandler) Delete(c *gin.Context) {
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
