package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/config/gemini_api"
	"github.com/tnqbao/financial_management_bot/utils"
	"gorm.io/gorm"
	"strings"
	"time"
)

func ParseAddPaidMessagesWithGemini(inputText string, chatID int64) ([]utils.Payment, error) {
	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60) // UTC +7 giờ
	now := time.Now().In(loc)

	prompt := fmt.Sprintf(`
	(Văn bản: "%s") với (thời điểm hiện tại: %s)
	Phân tích đoạn văn trên và trích xuất thông tin chi tiêu theo đúng JSON chuẩn:
	[
		{
			"chat_id": %d,
			"category": "danh mục (VD: Thực phẩm, Dịch vụ, Giải trí)",
			"object": "mặt hàng (VD: Bánh mì, Tiền điện, Vé xem phim)",
			"price": số tiền (VD: 20.0, 100.5, 35k => 20000.0 , 100500.0 , 35000.0),
			"datePaid": "YYYY-MM-DDTHH:MM:SSZ" (ISO 8601, mặc định là thời điểm hiện tại)
		},
		{
			"chat_id": %d,
			"category": "danh mục (VD: Thực phẩm, Dịch vụ, Giải trí)",
			"object": "mặt hàng (VD: Phở, Tiền nước, Đi xem phim)",
			"price": số tiền (VD: 20.0, 100.5, 35k => 20000.0 , 100500.0 , 35000.0),
			"datePaid": "YYYY-MM-DDTHH:MM:SSZ" (ISO 8601, mặc định là thời điểm hiện tại)
		}
	]
	Lưu ý: Chỉ trả về **DUY NHẤT** JSON, không có bất kỳ văn bản nào khác!
`, inputText, now.Format("2006-01-02"), chatID, chatID)

	aiClient := gemini_api.NewAIClient()
	resp, err := aiClient.GetResponse(prompt)
	if err != nil {
		return nil, err
	}

	resp = strings.TrimSpace(resp)
	resp = strings.Trim(resp, "```json")
	resp = strings.Trim(resp, "```")
	resp = strings.Trim(resp, "`")

	var payments []utils.Payment

	err = json.Unmarshal([]byte(resp), &payments)
	if err != nil {
		return nil, fmt.Errorf("Lỗi xử lý JSON: %v", err)
	}

	for i := range payments {
		if payments[i].DatePaid.IsZero() {
			payments[i].DatePaid = time.Now().In(loc)
		}
	}

	return payments, nil
}

func HandlePaymentMessage(ctx context.Context, bot *tgbotapi.BotAPI, messageText string, chatID int64) {

	payments, err := ParseAddPaidMessagesWithGemini(messageText, chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Lỗi xử lý tin nhắn: "+err.Error())
		bot.Send(msg)
		return
	}

	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Lỗi kết nối cơ sở dữ liệu"))
		return
	}

	var confirmMsg strings.Builder
	confirmMsg.WriteString("✅ Đã lưu các khoản thanh toán:\n")
	for _, payment := range payments {
		confirmMsg.WriteString(fmt.Sprintf(
			"📌 Danh mục: %s\n🛒 Mặt hàng: %s\n💰 Giá: %.2f\n📅 Ngày: %s\n\n",
			payment.Category, payment.Object, payment.Price, payment.DatePaid.Format("02-01-2006"),
		))
	}

	bot.Send(tgbotapi.NewMessage(chatID, confirmMsg.String()))

	tx := db.Begin()
	for _, payment := range payments {
		if err := tx.Create(&payment).Error; err != nil {
			tx.Rollback()
			msg := tgbotapi.NewMessage(chatID, "Lỗi lưu dữ liệu: "+err.Error())
			bot.Send(msg)
			return
		}
	}
	tx.Commit()
}
