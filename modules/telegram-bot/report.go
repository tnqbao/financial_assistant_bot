package telegram_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func EscapeMarkdownV2(text string) string {
	specialChars := "_*[]()~`>#+-=|{}.!"
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, string(char), "\\"+string(char))
	}
	return text
}

func HandleSumReport(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, period string, startDate, endDate string) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi kết nối cơ sở dữ liệu"))
		return
	}

	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60) // UTC +7 giờ
	now := time.Now().In(loc)

	var start, end time.Time
	var periodLabel string

	// Xác định khoảng thời gian và label
	switch period {
	case "day":
		start = now.Truncate(24 * time.Hour)
		end = start.Add(24 * time.Hour)
		periodLabel = fmt.Sprintf("Từ ngày %s đến ngày %s", start.Format("02-01-2006"), end.Format("02-01-2006"))
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0)
		periodLabel = fmt.Sprintf("Từ ngày %s đến ngày %s", start.Format("02-01-2006"), end.Add(-time.Second).Format("02-01-2006"))
	case "year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		end = start.AddDate(1, 0, 0)
		periodLabel = fmt.Sprintf("Từ ngày %s đến ngày %s", start.Format("02-01-2006"), end.Add(-time.Second).Format("02-01-2006"))
	case "week":
		// Tính toán tuần (từ chủ nhật đến thứ 7)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -weekday+1) // Chủ nhật
		end = start.AddDate(0, 0, 7)          // Kết thúc vào thứ 7
		periodLabel = fmt.Sprintf("Từ ngày %s đến ngày %s", start.Format("02-01-2006"), end.Add(-time.Second).Format("02-01-2006"))
	case "custom":
		// Ngày bắt đầu -> Ngày kết thúc (tùy chỉnh)
		var err error
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi định dạng ngày bắt đầu"))
			return
		}
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi định dạng ngày kết thúc"))
			return
		}
		periodLabel = fmt.Sprintf("Từ ngày %s đến ngày %s", start.Format("02-01-2006"), end.Format("02-01-2006"))
	default:
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Không xác định được khoảng thời gian"))
		return
	}

	var total float64
	err := db.Model(&utils.Payment{}).
		Where("chat_id = ? AND date_paid >= ? AND date_paid < ?", chatID, start, end).
		Select("COALESCE(SUM(price), 0)").
		Scan(&total).Error

	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi truy vấn dữ liệu"))
		return
	}

	// Lấy các thông tin đối tượng, giá trị và thời gian
	type PaymentDetail struct {
		Object   string
		Price    float64
		DatePaid time.Time
	}
	var paymentDetails []PaymentDetail

	err = db.Model(&utils.Payment{}).
		Where("chat_id = ? AND date_paid >= ? AND date_paid < ?", chatID, start, end).
		Select("object, price, date_paid").
		Order("date_paid DESC").
		Scan(&paymentDetails).Error

	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi khi lấy danh sách chi tiêu"))
		return
	}

	// Tạo file Excel
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.NewSheet(sheet)

	// Thêm tiêu đề
	f.SetCellValue(sheet, "A1", "Dịch vụ")
	f.SetCellValue(sheet, "B1", "Số tiền (VND)")
	f.SetCellValue(sheet, "C1", "Thời gian")

	// Thêm tổng chi tiêu
	f.SetCellValue(sheet, "E1", "Tổng cộng")
	f.SetCellValue(sheet, "F1", fmt.Sprintf("%.2f", total))

	// Thêm dữ liệu chi tiêu
	row := 2
	for _, payment := range paymentDetails {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), payment.Object)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), payment.Price)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), payment.DatePaid.Format("02-01-2006"))
		row++
	}

	fileName := fmt.Sprintf("report_%s.xlsx", period)
	err = f.SaveAs(fileName)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi khi lưu file Excel"))
		return
	}

	periodMap := map[string]string{"day": "ngày", "month": "tháng", "year": "năm", "week": "tuần", "custom": "khoảng thời gian tùy chỉnh"}
	periodName := periodMap[period]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 *Báo cáo chi tiêu trong %s*\n", EscapeMarkdownV2(periodName)))
	sb.WriteString(fmt.Sprintf("🗓️ *Khoảng thời gian*: %s\n", EscapeMarkdownV2(periodLabel)))
	sb.WriteString(fmt.Sprintf("💰 *Tổng cộng*: %s VND\n\n", EscapeMarkdownV2(fmt.Sprintf("%.2f", total))))
	sb.WriteString("*Chi tiết theo danh mục:*\n")
	sb.WriteString("```plaintext\n")
	sb.WriteString(fmt.Sprintf("%-20s %-10s %-10s\n", "📌 Dịch vụ", "💵 Số tiền", "🔢 Thời gian"))
	sb.WriteString(strings.Repeat("-", 45) + "\n")

	for _, summary := range paymentDetails {
		sb.WriteString(fmt.Sprintf("%-20s %-10.2f %-10s\n", EscapeMarkdownV2(summary.Object), summary.Price, summary.DatePaid.Format("02-01-2006")))
	}

	sb.WriteString("```\n")

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)

	// Gửi file Excel cho người dùng
	file := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(fileName))
	bot.Send(file)
}
