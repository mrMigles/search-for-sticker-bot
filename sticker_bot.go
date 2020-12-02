package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"
)

type StickerBot struct {
	Bot      *tgbotapi.BotAPI
	Stickers *StickerResource
}

func NewStickerBot(token string, stickers *StickerResource) (StickerBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return StickerBot{}, err
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return StickerBot{
		Bot:      bot,
		Stickers: stickers,
	}, nil
}

func (s StickerBot) startBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := s.Bot.GetUpdatesChan(u)
	if err != nil {

	}

	for update := range updates {
		if update.Message != nil {
			s.handleMessage(update.Message)
		}
		if update.InlineQuery != nil {
			s.handleInline(update.InlineQuery)
		}
	}
}

func (s StickerBot) handleMessage(message *tgbotapi.Message) {
	user, err := s.Stickers.FindUser(message.From.ID)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	if user == nil {
		user = &User{UserId: message.From.ID, Private: false}
		err := s.Stickers.SaveUser(user)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
	}

	if message.Text != "" &&
		message.ReplyToMessage != nil &&
		message.ReplyToMessage.Sticker != nil {

		sticker := message.ReplyToMessage.Sticker

		text := trim(message.Text, 100)

		foundStickers := s.Stickers.FindPublicStickersByFileIdAndUser(sticker.FileUniqueID, *user)
		var stickerR *Sticker
		if len(foundStickers) > 0 {
			stickerR = &foundStickers[0]
			stickerR.Text = text
		} else {
			stickerR = &Sticker{
				FileId:       sticker.FileID,
				UniqueFileId: sticker.FileUniqueID,
				Text:         text,
				Pack:         sticker.SetName,
				AddedBy:      message.From.ID,
				Emoji:        sticker.Emoji,
			}
		}

		if err := s.Stickers.SaveSticker(stickerR); err != nil {
			log.Printf("Error: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Что то пошло не так")
			if _, err := s.Bot.Send(msg); err != nil {
				log.Printf("Error: %v", err)
			}
			return
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, "Стикер сохранен. Поиск по ключевым словам: "+text)
		msg.ReplyToMessageID = message.ReplyToMessage.MessageID
		if _, err := s.Bot.Send(msg); err != nil {
			log.Printf("Error: %v", err)
		}
		return
	}

	if message.Sticker != nil {
		foundStickers := s.Stickers.FindPublicStickersByFileIdAndUser(message.Sticker.FileUniqueID, *user)
		if len(foundStickers) > 0 {
			sticker := &foundStickers[0]
			msg := tgbotapi.NewMessage(message.Chat.ID, "Ты уже добавлял этот стикер с подписью: "+sticker.Text)
			msg.ReplyToMessageID = message.MessageID
			if _, err := s.Bot.Send(msg); err != nil {
				log.Printf("Error: %v", err)
			}
		}

		if !user.Private {
			foundStickers = s.Stickers.FindPublicStickersByFileId(message.Sticker.FileUniqueID)
			var stickerTexts []string
			for _, sticker := range foundStickers {
				if sticker.AddedBy != user.UserId {
					stickerTexts = append(stickerTexts, sticker.Text)
				}
			}
			if len(stickerTexts) > 0 {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Другие пользователи уже добавляли данный стикер с подписями: "+strings.Join(stickerTexts, ", "))
				msg.ReplyToMessageID = message.MessageID
				if _, err := s.Bot.Send(msg); err != nil {
					log.Printf("Error: %v", err)
				}
			}
		}
		return
	}

	if strings.Contains(message.Text, "/start") {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Привет! Чтобы добавить стикер просто отправь его мне и реплай с описанием для поиска. "+
			"\n\n/help - для справки.")
		if _, err := s.Bot.Send(msg); err != nil {
			log.Printf("Error: %v", err)
		}
		return
	}

	if strings.Contains(message.Text, "/public") {
		if user.Private {
			user.Private = false
			err := s.Stickers.ChangeUserType(user)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			msg := tgbotapi.NewMessage(message.Chat.ID, "Переключил на публичный режим, теперь ты можешь искать по всем добавленным стикерам. "+
				"\nДля перехода в приватный режим используй /private")
			if _, err := s.Bot.Send(msg); err != nil {
				log.Printf("Error: %v", err)
			}
		}
		return
	}

	if strings.Contains(message.Text, "/private") {
		if !user.Private {
			user.Private = true
			err := s.Stickers.ChangeUserType(user)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			msg := tgbotapi.NewMessage(message.Chat.ID, "Переключил на приватный режим, теперь ты можешь искать только по добавленным тобой стикерам. "+
				"\nДля перехода в публичный режим используй /public")
			if _, err := s.Bot.Send(msg); err != nil {
				log.Printf("Error: %v", err)
			}
		}
		return
	}

	if strings.Contains(message.Text, "/help") {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Отправь мне стикер и реплай с описанием для поиска и я добавлю его в базу. "+
			"\nМожешь использовать несколько тегов для описания стикера, чтобы улушить поиск (но не больше 100 символов)."+
			"\nЧтобы искать просто введи `@"+s.Bot.Self.UserName+" текст для поиска стикера`."+
			"\n\n/public - чтобы искать по стикерам, добавленным другими пользователями. Твои стикеры в поиске будут идти первыми. (включен по умолчанию)."+
			"\n/private - чтобы искать только по своим стикерам. Если ты этого так хочешь.")
		if _, err := s.Bot.Send(msg); err != nil {
			log.Printf("Error: %v", err)
		}
		return
	}

	if message.Text != "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала отправь мне стикер, а потом в реплай на сообщение со стикером отправь текст для поиска.")
		if _, err := s.Bot.Send(msg); err != nil {
			log.Printf("Error: %v", err)
		}
		return
	}
}

func (s StickerBot) handleInline(inline *tgbotapi.InlineQuery) {
	if len(inline.Query) > 2 {
		user, err := s.Stickers.FindUser(inline.From.ID)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		if user == nil {
			user = &User{UserId: inline.From.ID, Private: false}
			err := s.Stickers.SaveUser(user)
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}
		}

		var stickers []interface{}
		foundStickers := s.Stickers.FindStickersByTextAndUser(inline.Query, *user)
		for i, sticker := range foundStickers {
			stickers = append(stickers, tgbotapi.InlineQueryResultCachedSticker{
				Type:      "sticker",
				ID:        strconv.Itoa(i),
				StickerID: sticker.FileId,
				Title:     sticker.Text,
			})
		}

		inlineConfig := tgbotapi.InlineConfig{
			InlineQueryID:     inline.ID,
			Results:           stickers,
			IsPersonal:        user.Private,
			SwitchPMText:      "Добавить новый",
			SwitchPMParameter: "add_sticker",
		}
		_, err = s.Bot.AnswerInlineQuery(inlineConfig)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}

}

func trim(s string, length int) string {
	var size, x int

	for i := 0; i < length && x < len(s); i++ {
		_, size = utf8.DecodeRuneInString(s[x:])
		x += size
	}

	return s[:x]
}
