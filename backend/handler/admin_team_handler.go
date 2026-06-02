package handler

import (
	"exchange_cdk/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminTeamHandler struct {
	svc *service.TeamService
}

func NewAdminTeamHandler(svc *service.TeamService) *AdminTeamHandler {
	return &AdminTeamHandler{svc: svc}
}

type joinTeamReq struct {
	OwnerUsername string `json:"owner_username" binding:"required"`
}

// MyTeam GET /api/admin/teams/my
func (h *AdminTeamHandler) MyTeam(c *gin.Context) {
	team, err := h.svc.MyTeam(getUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": team})
}

// JoinedTeams GET /api/admin/teams/joined
func (h *AdminTeamHandler) JoinedTeams(c *gin.Context) {
	teams, err := h.svc.JoinedTeams(getUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"list": teams}})
}

// Join POST /api/admin/teams/join
func (h *AdminTeamHandler) Join(c *gin.Context) {
	var req joinTeamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "请输入团队拥有者用户名", "data": nil})
		return
	}
	team, err := h.svc.JoinByOwnerUsername(getUserID(c), req.OwnerUsername)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": team})
}

// RemoveMember DELETE /api/admin/teams/members/:member_id
func (h *AdminTeamHandler) RemoveMember(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("member_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "成员ID无效", "data": nil})
		return
	}
	if err := h.svc.RemoveMember(getUserID(c), uint(memberID)); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
}
