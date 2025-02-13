package telegram_bot

import (
	"context"
	"fmt"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func HandleExportExcel(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, startDateStr, endDateStr string) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi kết nối cơ sở dữ liệu"))
		return
	}

	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, loc)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Ngày bắt đầu không hợp lệ, định dạng đúng: YYYY-MM-DD"))
		return
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, loc)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Ngày kết thúc không hợp lệ, định dạng đúng: YYYY-MM-DD"))
		return
	}
	endDate = endDate.Add(24 * time.Hour)

	// Lấy dữ liệu từ database
	type Payment struct {
		DatePaid time.Time
		Object   string
		Price    float64
	}
	var payments []Payment

	err = db.Model(&utils.Payment{}).
		Where("chat_id = ? AND date_paid >= ? AND date_paid < ?", chatID, startDate, endDate).
		Select("date_paid, object, price").
		Order("date_paid ASC").
		Scan(&payments).Error

	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi truy vấn dữ liệu"))
		return
	}

	if len(payments) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "📭 Không có dữ liệu chi tiêu trong khoảng thời gian này."))
		return
	}

	// Tạo file Excel
	file := excelize.NewFile()
	sheetName := "Chi tiêu"
	file.SetSheetName("Sheet1", sheetName)

	// Ghi tiêu đề cột
	headers := []string{"Ngày", "Dịch vụ", "Số tiền (VND)"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		file.SetCellValue(sheetName, cell, h)
	}

	// Ghi dữ liệu vào Excel
	for i, p := range payments {
		file.SetCellValue(sheetName, fmt.Sprintf("A%d", i+2), p.DatePaid.Format("2006-01-02"))
		file.SetCellValue(sheetName, fmt.Sprintf("B%d", i+2), p.Object)
		file.SetCellValue(sheetName, fmt.Sprintf("C%d", i+2), p.Price)
	}

	// Lưu file tạm thời
	filePath := fmt.Sprintf("chi_tieu_%s_den_%s.xlsx", startDateStr, endDateStr)
	err = file.SaveAs(filePath)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi khi tạo file Excel"))
		return
	}
	defer os.Remove(filePath) // Xóa file sau khi gửi

	// Gửi file Excel về Telegram
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	doc.Caption = fmt.Sprintf("📂 Báo cáo chi tiêu từ %s đến %s", startDateStr, endDateStr)
	bot.Send(doc)
}
