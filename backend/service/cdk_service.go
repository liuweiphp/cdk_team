package service

import (
	"bufio"
	"exchange_cdk/model"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CdkService struct{ db *gorm.DB }

func NewCdkService(db *gorm.DB) *CdkService { return &CdkService{db: db} }

const maxFileSize = 5 << 20
const maxLines = 50000
const batchSize = 500

// InvalidRow 无效行
type InvalidRow struct {
	Line   int    `json:"line"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

type CodeItemMapping struct {
	Line     int
	Code     string
	Filename string
}

// Import 导入兑换码文件：校验→解析→去重→批量 INSERT IGNORE
func (s *CdkService) Import(file multipart.File, header *multipart.FileHeader, itemID uint, remark string, createdBy uint) (*model.CdkImport, []InvalidRow, error) {
	if header.Size > maxFileSize {
		return nil, nil, fmt.Errorf("文件大小超过上限 5MB")
	}
	var item model.RedeemItem
	if err := s.db.Where("id = ? AND status = 'active'", itemID).First(&item).Error; err != nil {
		return nil, nil, fmt.Errorf("兑换内容不存在或已禁用")
	}

	codes, invalids, err := parseFile(file, header.Filename)
	if err != nil {
		return nil, nil, err
	}
	if len(codes) > maxLines {
		return nil, nil, fmt.Errorf("文件行数超过上限 %d", maxLines)
	}

	// 文件内去重
	seen := make(map[string]bool, len(codes))
	uniqCodes := make([]string, 0, len(codes))
	for _, c := range codes {
		norm := normalizeCode(c)
		if seen[norm] {
			continue
		}
		seen[norm] = true
		uniqCodes = append(uniqCodes, norm)
	}

	imp := &model.CdkImport{
		Filename:  header.Filename,
		Amount:    0,
		ItemID:    &itemID,
		Total:     uint(len(codes)),
		Invalid:   uint(len(invalids)),
		Remark:    remark,
		CreatedBy: createdBy,
	}
	if err := s.db.Create(imp).Error; err != nil {
		return nil, nil, err
	}

	inserted := 0
	skipped := 0
	for i := 0; i < len(uniqCodes); i += batchSize {
		end := i + batchSize
		if end > len(uniqCodes) {
			end = len(uniqCodes)
		}
		batch := make([]model.Cdk, 0, end-i)
		for _, code := range uniqCodes[i:end] {
			batch = append(batch, model.Cdk{
				Code:     code,
				Amount:   0,
				ItemID:   &itemID,
				Status:   "unused",
				ImportID: imp.ID,
			})
		}
		result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		inserted += int(result.RowsAffected)
		skipped += len(batch) - int(result.RowsAffected)
	}

	imp.Inserted = uint(inserted)
	imp.Skipped = uint(skipped)
	s.db.Save(imp)

	return imp, invalids, nil
}

// ImportMappings 导入兑换码与文件名映射,每行格式: 兑换码,文件名
func (s *CdkService) ImportMappings(file multipart.File, header *multipart.FileHeader, remark string, createdBy uint) (*model.CdkImport, []InvalidRow, error) {
	if header.Size > maxFileSize {
		return nil, nil, fmt.Errorf("文件大小超过上限 5MB")
	}
	mappings, invalids, err := parseCodeItemMappings(file, header.Filename)
	if err != nil {
		return nil, nil, err
	}
	if len(mappings) > maxLines {
		return nil, nil, fmt.Errorf("文件行数超过上限 %d", maxLines)
	}
	totalRows := len(mappings) + len(invalids)

	var items []model.RedeemItem
	if err := s.db.Where("status = 'active'").Find(&items).Error; err != nil {
		return nil, nil, err
	}
	itemByFilename := make(map[string]uint, len(items))
	allItemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemByFilename[normalizeFilename(item.Filename)] = item.ID
		allItemIDs = append(allItemIDs, item.ID)
	}
	usedItemIDs := make(map[uint]bool)
	if len(allItemIDs) > 0 {
		var existingItemIDs []uint
		if err := s.db.Model(&model.Cdk{}).Where("item_id IN ?", allItemIDs).Distinct().Pluck("item_id", &existingItemIDs).Error; err != nil {
			return nil, nil, err
		}
		for _, id := range existingItemIDs {
			usedItemIDs[id] = true
		}
	}

	seen := make(map[string]bool, len(mappings))
	seenItems := make(map[uint]bool, len(mappings))
	validMappings := make([]CodeItemMapping, 0, len(mappings))
	itemIDs := make(map[string]uint, len(mappings))
	for _, m := range mappings {
		if seen[m.Code] {
			continue
		}
		seen[m.Code] = true
		itemID, ok := itemByFilename[normalizeFilename(m.Filename)]
		if !ok {
			invalids = append(invalids, InvalidRow{Line: m.Line, Code: m.Code, Reason: "文件名未匹配到已上传内容: " + m.Filename})
			continue
		}
		if usedItemIDs[itemID] {
			invalids = append(invalids, InvalidRow{Line: m.Line, Code: m.Code, Reason: "该文件已绑定兑换码: " + m.Filename})
			continue
		}
		if seenItems[itemID] {
			invalids = append(invalids, InvalidRow{Line: m.Line, Code: m.Code, Reason: "同一文件只能绑定一个兑换码: " + m.Filename})
			continue
		}
		seenItems[itemID] = true
		validMappings = append(validMappings, m)
		itemIDs[m.Code] = itemID
	}

	imp := &model.CdkImport{
		Filename:  header.Filename,
		Amount:    0,
		Total:     uint(totalRows),
		Invalid:   uint(len(invalids)),
		Remark:    remark,
		CreatedBy: createdBy,
	}
	if err := s.db.Create(imp).Error; err != nil {
		return nil, nil, err
	}

	inserted := 0
	skipped := 0
	for i := 0; i < len(validMappings); i += batchSize {
		end := i + batchSize
		if end > len(validMappings) {
			end = len(validMappings)
		}
		batch := make([]model.Cdk, 0, end-i)
		for _, m := range validMappings[i:end] {
			itemID := itemIDs[m.Code]
			batch = append(batch, model.Cdk{
				Code:     m.Code,
				Amount:   0,
				ItemID:   &itemID,
				Status:   "unused",
				ImportID: imp.ID,
			})
		}
		result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		inserted += int(result.RowsAffected)
		skipped += len(batch) - int(result.RowsAffected)
	}

	imp.Inserted = uint(inserted)
	imp.Skipped = uint(skipped)
	s.db.Save(imp)
	return imp, invalids, nil
}

// List 当前用户可见的 CDK 管理列表
func (s *CdkService) List(page, pageSize int, amount float64, status, code string, importID, itemID uint, currentUserID uint) ([]model.Cdk, int64, error) {
	var list []model.Cdk
	var total int64
	ownerIDs, err := accessibleOwnerIDs(s.db, currentUserID)
	if err != nil {
		return nil, 0, err
	}
	q := s.db.Model(&model.Cdk{}).
		Preload("RedeemItem").
		Preload("RedeemItem.Creator").
		Joins("JOIN cdk_imports ON cdk_imports.id = cdks.import_id").
		Joins("LEFT JOIN redeem_items ON redeem_items.id = cdks.item_id AND redeem_items.deleted_at IS NULL").
		Where("COALESCE(redeem_items.created_by, cdk_imports.created_by) IN ?", ownerIDs)
	if amount > 0 {
		q = q.Where("cdks.amount = ?", amount)
	}
	if itemID > 0 {
		q = q.Where("cdks.item_id = ?", itemID)
	}
	if status != "" {
		q = q.Where("cdks.status = ?", status)
	}
	if code != "" {
		q = q.Where("cdks.code LIKE ?", "%"+code+"%")
	}
	if importID > 0 {
		q = q.Where("cdks.import_id = ?", importID)
	}
	q.Count(&total)
	if err := q.Order("cdks.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ImportList 导入记录列表
func (s *CdkService) ImportList(page, pageSize int, amount float64) ([]model.CdkImport, int64, error) {
	var list []model.CdkImport
	var total int64
	q := s.db.Model(&model.CdkImport{}).Preload("RedeemItem")
	if amount > 0 {
		q = q.Where("amount = ?", amount)
	}
	q.Count(&total)
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func parseFile(file multipart.File, filename string) (codes []string, invalids []InvalidRow, err error) {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xls") {
		return parseExcel(file)
	}
	return parseCSV(file)
}

func parseCodeItemMappings(file io.Reader, filename string) ([]CodeItemMapping, []InvalidRow, error) {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xls") {
		mf, ok := file.(multipart.File)
		if !ok {
			return nil, nil, fmt.Errorf("无法读取 Excel 文件")
		}
		return parseExcelMappings(mf)
	}
	return parseTextMappings(file)
}

func parseTextMappings(file io.Reader) ([]CodeItemMapping, []InvalidRow, error) {
	var mappings []CodeItemMapping
	var invalids []InvalidRow
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		code, filename, ok := splitMappingLine(line)
		if !ok {
			invalids = append(invalids, InvalidRow{Line: lineNo, Code: line, Reason: "格式应为: 兑换码,文件名"})
			continue
		}
		if isMappingHeader(code, filename) {
			continue
		}
		mapping, invalid := validateMapping(lineNo, code, filename)
		if invalid != nil {
			invalids = append(invalids, *invalid)
			continue
		}
		mappings = append(mappings, mapping)
	}
	return mappings, invalids, scanner.Err()
}

func parseExcelMappings(file multipart.File) ([]CodeItemMapping, []InvalidRow, error) {
	var mappings []CodeItemMapping
	var invalids []InvalidRow
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("无法读取 Excel 文件: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, nil, err
	}
	for i, row := range rows {
		lineNo := i + 1
		if len(row) == 0 {
			continue
		}
		code := strings.TrimSpace(row[0])
		filename := ""
		if len(row) > 1 {
			filename = strings.TrimSpace(row[1])
		}
		if code == "" && filename == "" {
			continue
		}
		if isMappingHeader(code, filename) {
			continue
		}
		mapping, invalid := validateMapping(lineNo, code, filename)
		if invalid != nil {
			invalids = append(invalids, *invalid)
			continue
		}
		mappings = append(mappings, mapping)
	}
	return mappings, invalids, nil
}

func splitMappingLine(line string) (string, string, bool) {
	sep := ","
	if strings.Contains(line, "\t") {
		sep = "\t"
	}
	parts := strings.SplitN(line, sep, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func validateMapping(lineNo int, code, filename string) (CodeItemMapping, *InvalidRow) {
	normCode := normalizeCode(code)
	filename = normalizeFilename(filename)
	if normCode == "" || filename == "" {
		return CodeItemMapping{}, &InvalidRow{Line: lineNo, Code: code, Reason: "兑换码和文件名不能为空"}
	}
	if len(normCode) < 8 || len(normCode) > 64 {
		return CodeItemMapping{}, &InvalidRow{Line: lineNo, Code: code, Reason: "长度不在8-64范围内"}
	}
	if !validCodeChars(normCode) {
		return CodeItemMapping{}, &InvalidRow{Line: lineNo, Code: code, Reason: "包含无效字符"}
	}
	return CodeItemMapping{Line: lineNo, Code: normCode, Filename: filename}, nil
}

func isMappingHeader(code, filename string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	filename = strings.ToLower(strings.TrimSpace(filename))
	return (code == "code" || code == "兑换码") && (filename == "filename" || filename == "文件名")
}

func parseCSV(file io.Reader) ([]string, []InvalidRow, error) {
	var codes []string
	var invalids []InvalidRow
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		code := strings.TrimSpace(scanner.Text())
		if code == "" {
			continue
		}
		norm := normalizeCode(code)
		if len(norm) < 8 || len(norm) > 64 {
			invalids = append(invalids, InvalidRow{Line: lineNo, Code: code, Reason: "长度不在8-64范围内"})
			continue
		}
		if !validCodeChars(norm) {
			invalids = append(invalids, InvalidRow{Line: lineNo, Code: code, Reason: "包含无效字符"})
			continue
		}
		codes = append(codes, norm)
	}
	return codes, invalids, scanner.Err()
}

func parseExcel(file multipart.File) ([]string, []InvalidRow, error) {
	var codes []string
	var invalids []InvalidRow
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("无法读取 Excel 文件: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, nil, err
	}
	for i, row := range rows {
		lineNo := i + 1
		if len(row) == 0 {
			continue
		}
		code := strings.TrimSpace(row[0])
		if code == "" {
			continue
		}
		norm := normalizeCode(code)
		if len(norm) < 8 || len(norm) > 64 {
			invalids = append(invalids, InvalidRow{Line: lineNo, Code: code, Reason: "长度不在8-64范围内"})
			continue
		}
		if !validCodeChars(norm) {
			invalids = append(invalids, InvalidRow{Line: lineNo, Code: code, Reason: "包含无效字符"})
			continue
		}
		codes = append(codes, norm)
	}
	return codes, invalids, nil
}

func validCodeChars(code string) bool {
	for _, r := range code {
		if !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789", r) {
			return false
		}
	}
	return true
}
