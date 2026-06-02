package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminRedeemItemHandler struct {
	svc *service.RedeemItemService
}

func NewAdminRedeemItemHandler(svc *service.RedeemItemService) *AdminRedeemItemHandler {
	return &AdminRedeemItemHandler{svc: svc}
}

type redeemItemReq struct {
	Name     string `json:"name" binding:"required"`
	Filename string `json:"filename"`
	Content  string `json:"content" binding:"required"`
	Status   string `json:"status"`
}

// List GET /api/admin/redeem-items
func (h *AdminRedeemItemHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	keyword := c.Query("keyword")
	status := c.Query("status")
	list, total, err := h.svc.List(page, pageSize, keyword, status, getUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// Create POST /api/admin/redeem-items
func (h *AdminRedeemItemHandler) Create(c *gin.Context) {
	var req redeemItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	item, err := h.svc.Create(req.Name, req.Filename, req.Content, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": item})
}

// ImportFiles POST /api/admin/redeem-items/import
func (h *AdminRedeemItemHandler) ImportFiles(c *gin.Context) {
	templateID := uint(queryInt(c, "template_id", 0))
	if templateID == 0 {
		var req struct {
			TemplateID uint `form:"template_id"`
		}
		_ = c.ShouldBind(&req)
		templateID = req.TemplateID
	}
	if templateID == 0 {
		c.JSON(400, gin.H{"code": 40001, "message": "请选择模板", "data": nil})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "请上传文件", "data": nil})
		return
	}
	file.Close()
	result, err := h.svc.ImportLines(header, templateID, getUserID(c))
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": result})
}

// Update PUT /api/admin/redeem-items/:id
func (h *AdminRedeemItemHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req redeemItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.Update(uint(id), req.Name, req.Filename, req.Content, req.Status, getUserID(c)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

// Delete DELETE /api/admin/redeem-items/:id
func (h *AdminRedeemItemHandler) Delete(c *gin.Context) {
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
