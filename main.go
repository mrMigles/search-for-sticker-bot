package main

import (
	"log"
	"os"
	"strings"
	"sync"
)

type Bot interface {
	startBot()
}

var botTokens = GetEnv("BOT_TOKENS", "")

func main() {
	stickerResource := NewStickerResource()
	tokens := strings.Split(botTokens, ",")

	var wg sync.WaitGroup
	wg.Add(len(tokens))

	for _, token := range tokens {
		stickerBot, err := NewStickerBot(token, stickerResource)
		if err != nil {
			log.Printf("Error init bot: %v", err)
		} else {
			go stickerBot.startBot()
		}
	}

	wg.Wait()
	log.Fatal("There are no active bots")
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
