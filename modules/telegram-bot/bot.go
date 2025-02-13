package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tnqbao/financial_management_bot/config/gemini_api"
	"gorm.io/gorm"
)

type CustomDateRange struct {
	DateStart string `json:"dateStart"`
	DateEnd   string `json:"dateEnd"`
}

type APIResponse struct {
	Type      string `json:"type"`
	Period    string `json:"period,omitempty"`
	DateStart string `json:"dateStart,omitempty"`
	DateEnd   string `json:"dateEnd,omitempty"`
	Error     string `json:"error,omitempty"`
}

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
	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	now := time.Now().In(loc)
	aiClient := gemini_api.NewAIClient()
	prompt := fmt.Sprintf(`
  Xác định loại lệnh từ tin nhắn người dùng. Nếu tin nhắn thuộc một trong các loại sau:
  - Tổng chi tiêu ngày => Trả về: { "type": "sum", "period": "day" }
  - Tổng chi tiêu tuần => Trả về: { "type": "sum", "period": "week" }
  - Tổng chi tiêu tháng => Trả về: { "type": "sum", "period": "month" }
  - Tổng chi tiêu năm => Trả về: { "type": "sum", "period": "year" }
  - Tổng chi tiêu có ngày bắt đầu và kết thúc => Trả về: { "type": "custom", "dateStart": "YYYY-MM-DD", "dateEnd": "YYYY-MM-DD" }
  - Tin nhắn kèm giá tiền hoặc mô ta => Trả về: { "type": "other" }
  🚫 Lưu ý:
  - Chỉ in đúng một giá trị trong 5 giá trị trên, không thêm bất cứ ký tự hay văn bản nào khác.
  - Nếu tin nhắn không thuộc bất kỳ loại nào trên, chỉ in: { "type": "other" }
  - Nếu là lệnh !custom, trả về kết quả dưới dạng JSON với 2 trường 'dateStart' và 'dateEnd' (YYYY-MM-DD)

  🔹 Tin nhắn: "%s"
  * Ngày hiện tại: %s
`, messageText, now.Format("2006-01-02"))

	detectCase, err := aiClient.GetResponse(prompt)
	if err != nil {
		log.Printf("❌ Lỗi khi gọi AI: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Lỗi xử lý tin nhắn, vui lòng thử lại sau."))
		return
	}
	detectCase = strings.TrimSpace(detectCase)
	detectCase = strings.Trim(detectCase, "```json")
	detectCase = strings.Trim(detectCase, "```")
	detectCase = strings.Trim(detectCase, "`")
	fmt.Println(detectCase)
	var apiResponse APIResponse
	err = json.Unmarshal([]byte(detectCase), &apiResponse)
	if err != nil {
		log.Printf("❌ Lỗi khi phân tích JSON: %v", err)
		apiResponse = APIResponse{
			Error: "Lỗi trong cú pháp tin nhắn từ AI, vui lòng thử lại.",
		}
	}

	if apiResponse.Error != "" {
		responseJSON, _ := json.Marshal(apiResponse)
		bot.Send(tgbotapi.NewMessage(chatID, string(responseJSON)))
		return
	}

	switch apiResponse.Type {
	case "sum":
		switch apiResponse.Period {
		case "day":
			HandleSumReport(ctx, bot, chatID, "day", "", "")
		case "month":
			HandleSumReport(ctx, bot, chatID, "month", "", "")
		case "week":
			HandleSumReport(ctx, bot, chatID, "week", "", "")
		case "year":
			HandleSumReport(ctx, bot, chatID, "year", "", "")
		default:
			bot.Send(tgbotapi.NewMessage(chatID, "⚠ Lệnh tổng chi tiêu không hợp lệ"))
		}
	case "custom":
		if apiResponse.DateStart != "" && apiResponse.DateEnd != "" {
			HandleSumReport(ctx, bot, chatID, "custom", apiResponse.DateStart, apiResponse.DateEnd)
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠ Lệnh tùy chỉnh thiếu thông tin ngày bắt đầu và kết thúc"))
		}
	case "other":
		HandlePaymentMessage(ctx, bot, messageText, chatID)
	default:
		log.Printf("⚠ AI trả về kết quả không hợp lệ: %s", apiResponse.Type)
		bot.Send(tgbotapi.NewMessage(chatID, "⚠ Tôi là trợ thủ quản lý chi tiêu, chúc bạn một ngày tốt lành!"))
	}
}
