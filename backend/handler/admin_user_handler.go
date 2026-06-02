package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminUserHandler struct {
	svc *service.UserService
}

func NewAdminUserHandler(svc *service.UserService) *AdminUserHandler {
	return &AdminUserHandler{svc: svc}
}

type createUserReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type updateUserReq struct {
	Status   *string `json:"status"`
	Role     *string `json:"role"`
	Password *string `json:"password"`
}

// List GET /api/admin/users
func (h *AdminUserHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	keyword := c.Query("keyword")
	status := c.Query("status")

	list, total, err := h.svc.List(page, pageSize, keyword, status)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}

// Create POST /api/admin/users
func (h *AdminUserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "user"
	}
	user, err := h.svc.Create(req.Username, req.Password, req.Role)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": user})
}

// Update PATCH /api/admin/users/:id
func (h *AdminUserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "ID无效", "data": nil})
		return
	}
	var req updateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	if err := h.svc.Update(uint(id), req.Status, req.Role, req.Password); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}
