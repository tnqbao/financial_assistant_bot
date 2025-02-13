package telegram_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/utils"
	"gorm.io/gorm"
)

func EscapeMarkdownV2(text string) string {
	specialChars := "_*[]()~`>#+-=|{}.!"
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, string(char), "\\"+string(char))
	}
	return text
}

func HandleSumReport(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, period string) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi kết nối cơ sở dữ liệu"))
		return
	}

	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60) // UTC +7 giờ
	now := time.Now().In(loc)

	var start, end time.Time

	switch period {
	case "day":
		start = now.Truncate(24 * time.Hour)
		end = start.Add(24 * time.Hour)
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0)
	case "year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		end = start.AddDate(1, 0, 0)
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

	type CategorySummary struct {
		Object    string
		TotalCost float64
		Count     int
	}
	var categorySummaries []CategorySummary

	err = db.Model(&utils.Payment{}).
		Where("chat_id = ? AND date_paid >= ? AND date_paid < ?", chatID, start, end).
		Select("object, COUNT(*) as count, SUM(price) as total_cost").
		Group("object").
		Order("total_cost DESC").
		Scan(&categorySummaries).Error

	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi khi lấy danh sách chi tiêu theo danh mục"))
		return
	}

	periodMap := map[string]string{"day": "ngày", "month": "tháng", "year": "năm"}
	periodName := periodMap[period]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 *Báo cáo chi tiêu trong %s*\n", EscapeMarkdownV2(periodName)))
	sb.WriteString(fmt.Sprintf("💰 *Tổng cộng*: %s VND\n\n", EscapeMarkdownV2(fmt.Sprintf("%.2f", total))))
	sb.WriteString("*Chi tiết theo danh mục:*\n")
	sb.WriteString("```plaintext\n")
	sb.WriteString(fmt.Sprintf("%-20s %-10s %-10s\n", "📌 Dịch vụ", "💵 Số tiền", "🔢 Số lần"))
	sb.WriteString(strings.Repeat("-", 45) + "\n")

	for _, summary := range categorySummaries {
		sb.WriteString(fmt.Sprintf("%-20s %-10.2f %-10d\n", EscapeMarkdownV2(summary.Object), summary.TotalCost, summary.Count))
	}

	sb.WriteString("```\n")

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}
