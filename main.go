package main

import (
	"context"
	"github.com/tnqbao/financial_management_bot/config/gemini_api"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/tnqbao/financial_management_bot/config/database"
	telegram_bot "github.com/tnqbao/financial_management_bot/modules/telegram-bot"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠ Không tìm thấy file .env, sử dụng biến môi trường hệ thống")
	}

	gemini_api.LoadAPIKeys()
	db := database.InitDB()
	if db == nil {
		log.Fatal("❌ Lỗi kết nối database")
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "db", db)
	defer cancel()

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("❌ Chưa thiết lập TELEGRAM_BOT_TOKEN trong môi trường")
	}

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("❌ Lỗi khởi tạo bot Telegram:", err)
	}
	bot.Debug = true

	log.Println("🤖 Bot Telegram đã sẵn sàng, chờ tin nhắn...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for update := range updates {
			if update.Message != nil {
				go telegram_bot.HandleCommand(ctx, bot, update)
			}
		}
	}()

	<-sigChan
	log.Println("🛑 Bot đang tắt...")
	cancel()
}
