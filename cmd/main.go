package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"tanysu-bot/config"
	"tanysu-bot/internal/handler"
	"tanysu-bot/internal/repository"
	"tanysu-bot/traits/database"

	"github.com/go-telegram/bot"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	//chatState := handler.GetChatState()
	redisClient := database.RedisConnection(cfg)

	chatRedisState := repository.NewRedisClient(redisClient)

	handler := handler.NewHandler(chatRedisState)

	opts := []bot.Option{
		bot.WithCallbackQueryDataHandler("chat", bot.MatchTypePrefix, handler.ChatButtonHandler),
		bot.WithCallbackQueryDataHandler("select_", bot.MatchTypePrefix, handler.InlineHandler),
		bot.WithCallbackQueryDataHandler("exit", bot.MatchTypePrefix, handler.CallbackHandlerExit),
	}

	// Replace with your bot token
	token := cfg.Token
	// Create bot
	b, err := bot.New(token, opts...)
	if err != nil {
		fmt.Println("Error creating bot:", err)
		return
	}

	// 1) Регистрируем хендлер для обычных сообщений (пересылка между собеседниками)
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"",
		bot.MatchTypeContains,
		handler.MessageHandler,
	)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/hello", bot.MatchTypeExact, handler.HelloHandler)

	fmt.Println("Bot is running...")
	b.Start(ctx)
}
