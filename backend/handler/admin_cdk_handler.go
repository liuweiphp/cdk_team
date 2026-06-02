package handler

import (
	"exchange_cdk/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminCdkHandler struct {
	svc *service.CdkService
}

func NewAdminCdkHandler(svc *service.CdkService) *AdminCdkHandler {
	return &AdminCdkHandler{svc: svc}
}

type importReq struct {
	ItemID uint   `form:"item_id"`
	Remark string `form:"remark"`
}

// Import POST /api/admin/cdk/import
func (h *AdminCdkHandler) Import(c *gin.Context) {
	var req importReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "参数无效", "data": nil})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"code": 40001, "message": "请上传文件", "data": nil})
		return
	}
	defer file.Close()

	// 从 jwt claims 获取用户ID,这里简化为从 context 获取
	userID := getUserID(c)
	var invalids []service.InvalidRow
	var importErr error
	var importRecordID uint
	var total, inserted, skipped uint
	if req.ItemID > 0 {
		record, rows, err := h.svc.Import(file, header, req.ItemID, req.Remark, userID)
		invalids = rows
		importErr = err
		if record != nil {
			importRecordID = record.ID
			total = record.Total
			inserted = record.Inserted
			skipped = record.Skipped
		}
	} else {
		record, rows, err := h.svc.ImportMappings(file, header, req.Remark, userID)
		invalids = rows
		importErr = err
		if record != nil {
			importRecordID = record.ID
			total = record.Total
			inserted = record.Inserted
			skipped = record.Skipped
		}
	}
	if importErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": importErr.Error(), "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"import_id": importRecordID,
		"total":     total,
		"inserted":  inserted,
		"skipped":   skipped,
		"invalid":   invalids,
	}})
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

// ImportHistory GET /api/admin/cdk/imports
func (h *AdminCdkHandler) ImportHistory(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	amount := queryFloat(c, "amount", 0)

	list, total, err := h.svc.ImportList(page, pageSize, amount)
	if err != nil {
		c.JSON(500, gin.H{"code": 50001, "message": "查询失败", "data": nil})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
	}})
}
