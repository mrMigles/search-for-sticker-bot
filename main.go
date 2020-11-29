package main

import (
	"os"
)

type Bot interface {
	startBot()
}

var botToken = GetEnv("BOT_TOKEN", "")

func main() {
	stickerResource := NewStickerResource()
	stickerBot := NewStickerBot(botToken, stickerResource)

	stickerBot.startBot()
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
