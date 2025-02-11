package telegram_bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/config/gemini_api"
	"gorm.io/gorm"
)

func normalizeMessage(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.ReplaceAll(input, "\n", " ")
	return input
}

func HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageText := normalizeMessage(update.Message.Text)

	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi kết nối cơ sở dữ liệu"))
		return
	}

	aiClient := gemini_api.NewAIClient()
	prompt := fmt.Sprintf(`
		Xác định loại lệnh từ tin nhắn người dùng. Nếu tin nhắn thuộc một trong các loại sau:
		- Tổng chi tiêu ngày => Trả về: !sum day
		- Tổng chi tiêu tháng => Trả về: !sum month
		- Tổng chi tiêu năm => Trả về: !sum year
		
		🚫 Lưu ý:
		- Chỉ in đúng một giá trị trong ba giá trị trên, không thêm bất cứ ký tự hay văn bản nào khác.
		- Nếu tin nhắn không thuộc bất kỳ loại nào trên, chỉ in: !other
		
		🔹 Tin nhắn: "%s"
	`, messageText)

	detectCase, err := aiClient.GetResponse(prompt)
	if err != nil {
		log.Printf("❌ Lỗi khi gọi AI: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Lỗi xử lý tin nhắn, vui lòng thử lại sau."))
		return
	}

	switch strings.TrimSpace(detectCase) {
	case "!sum day":
		HandleSumReport(ctx, bot, chatID, "day")
	case "!sum month":
		HandleSumReport(ctx, bot, chatID, "month")
	case "!sum year":
		HandleSumReport(ctx, bot, chatID, "year")
	case "!other":
		HandlePaymentMessage(ctx, bot, messageText, chatID)
	default:
		log.Printf("⚠ AI trả về kết quả không hợp lệ: %s", detectCase)
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Không thể xử lý tin nhắn, vui lòng nhập đúng định dạng!"))
	}
}
