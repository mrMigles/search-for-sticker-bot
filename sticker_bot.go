package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strconv"
	"strings"
)

type StickerBot struct {
	Bot      *tgbotapi.BotAPI
	Stickers *StickerResource
}

func NewStickerBot(token string, stickers *StickerResource) StickerBot {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return StickerBot{
		Bot:      bot,
		Stickers: stickers,
	}
}

func (s StickerBot) startBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := s.Bot.GetUpdatesChan(u)
	if err != nil {

	}

	for update := range updates {
		if update.Message == nil && update.InlineQuery == nil {
			continue
		}

		if update.Message != nil &&
			update.Message.Text != "" &&
			update.Message.ReplyToMessage != nil &&
			update.Message.ReplyToMessage.Sticker != nil {
			sticker := update.Message.ReplyToMessage.Sticker
			text := update.Message.Text

			stickerR := &Sticker{
				FileId:  sticker.FileID,
				Text:    text,
				Pack:    sticker.SetName,
				AddedBy: update.Message.From.ID,
				Emoji:   sticker.Emoji,
			}

			if err := s.Stickers.SaveSticker(stickerR); err != nil {
				log.Printf("Error: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что то пошло не так")
				s.Bot.Send(msg)
				continue
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Стикер добавлен")
			s.Bot.Send(msg)
		}

		if update.Message != nil && strings.Contains(update.Message.Text, "/start") {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправь мне стикер и реплай с описанием, и я добавлю его в коллекцию для поиска.")
			s.Bot.Send(msg)
		}

		if update.InlineQuery != nil && len(update.InlineQuery.Query) > 2 {
			var stickers []interface{}
			foundStickers := s.Stickers.FindStickers(update.InlineQuery.Query)
			for i, sticker := range foundStickers {
				stickers = append(stickers, tgbotapi.InlineQueryResultCachedSticker{
					Type:      "sticker",
					ID:        strconv.Itoa(i),
					StickerID: sticker.FileId,
					Title:     sticker.Text,
				})
			}

			inlineConfig := tgbotapi.InlineConfig{
				InlineQueryID:     update.InlineQuery.ID,
				Results:           stickers,
				IsPersonal:        false,
				SwitchPMText:      "Добавить новый",
				SwitchPMParameter: "add_sticker",
			}
			_, err := s.Bot.AnswerInlineQuery(inlineConfig)
			if err != nil {
				log.Printf("Error %v", err)
			}
		}
	}
}
