package handler

import (
	"exchange_cdk/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// Overview GET /api/admin/stats/overview
func (h *StatsHandler) Overview(c *gin.Context) {
	data, err := h.svc.Overview()
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

// ByAmount GET /api/admin/stats/by-amount
func (h *StatsHandler) ByAmount(c *gin.Context) {
	data, err := h.svc.ByAmount()
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

// ByItem GET /api/admin/stats/by-item
func (h *StatsHandler) ByItem(c *gin.Context) {
	data, err := h.svc.ByItem()
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

// Daily GET /api/admin/stats/daily
func (h *StatsHandler) Daily(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")
	if start == "" || end == "" {
		c.JSON(400, gin.H{"code": 40001, "message": "start和end参数必填", "data": nil})
		return
	}
	data, err := h.svc.Daily(start, end)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

// TopUsers GET /api/admin/stats/top-users
func (h *StatsHandler) TopUsers(c *gin.Context) {
	limit := queryInt(c, "limit", 10)
	if limit > 100 {
		limit = 100
	}
	data, err := h.svc.TopUsers(limit)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

// Imports GET /api/admin/stats/imports
func (h *StatsHandler) Imports(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	data, total, err := h.svc.ImportList(page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": data, "total": total, "page": page, "page_size": pageSize,
	}})
}

// ByUserAmount GET /api/admin/stats/by-user-amount
func (h *StatsHandler) ByUserAmount(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")
	period := c.Query("period")
	if start == "" || end == "" || period == "" {
		c.JSON(400, gin.H{"code": 40001, "message": "start,end,period 参数必填", "data": nil})
		return
	}
	if period != "day" && period != "week" && period != "month" && period != "year" {
		c.JSON(400, gin.H{"code": 40001, "message": "period 必须为 day/week/month/year", "data": nil})
		return
	}
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}

	data, total, err := h.svc.ByUserAmount(period, start, end, page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": data, "total": total, "page": page, "page_size": pageSize,
	}})
}
