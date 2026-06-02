package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	svc *service.AnnouncementService
}

func NewAnnouncementHandler(svc *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

type announcementReq struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	IsPinned bool   `json:"is_pinned"`
}

// ListPublic GET /api/announcements
func (h *AnnouncementHandler) ListPublic(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	list, total, err := h.svc.ListPublic(page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// Create POST /api/admin/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req announcementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	userID := getUserID(c)
	a, err := h.svc.Create(req.Title, req.Content, req.IsPinned, userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "创建失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": a})
}

// Update PUT /api/admin/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req announcementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.Update(uint(id), req.Title, req.Content, req.IsPinned); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}

// Delete DELETE /api/admin/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "删除失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}
