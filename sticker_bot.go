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
		if update.ChosenInlineResult != nil {
			s.handleResult(update.ChosenInlineResult)
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

		set, err := s.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: message.Sticker.SetName})
		if err != nil {
			s.handleError(err, message)
		}
		stickerPack, err := s.Stickers.FindStickerPack(message.Sticker.SetName)
		if stickerPack == nil {

			err := s.Stickers.SaveStickerPack(&StickerPack{
				Name:        set.Name,
				Title:       set.Title,
				NumStickers: len(set.Stickers),
			})
			if err != nil {
				s.handleError(err, message)
			}

			err = s.Stickers.SaveStickersFromPack(convertTgStickersToLocal(set))
			if err != nil {
				s.handleError(err, message)
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

func (s StickerBot) handleError(err error, message *tgbotapi.Message) {
	log.Printf("Error: %v", err)
	msg := tgbotapi.NewMessage(message.Chat.ID, "Что то пошло не так")
	if _, err := s.Bot.Send(msg); err != nil {
		log.Printf("Error: %v", err)
	}
}

func (s StickerBot) handleInline(inline *tgbotapi.InlineQuery) {
	if len(inline.Query) > 0 {
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
		offset, err := strconv.Atoi(inline.Offset)
		if err != nil {
			offset = 0
		}
		for i, sticker := range foundStickers {
			if i < offset {
				continue
			}
			if i == offset+50 {
				break
			}
			stickers = append(stickers, tgbotapi.InlineQueryResultCachedSticker{
				Type:      "sticker",
				ID:        sticker.UniqueFileId,
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
			NextOffset:        strconv.Itoa(offset + 50),
		}
		_, err = s.Bot.AnswerInlineQuery(inlineConfig)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}

}

func (s StickerBot) handleResult(inline *tgbotapi.ChosenInlineResult) {
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

	user.Used = user.Used + 1
	err = s.Stickers.SaveUser(user)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	foundStickers := s.Stickers.FindStickersByFileId(inline.ResultID)
	for _, sticker := range foundStickers {
		sticker.Used = sticker.Used + 1
		err := s.Stickers.SaveSticker(&sticker)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
	}

	foundStickers = s.Stickers.FindPacksStickersByFileId(inline.ResultID)
	for _, sticker := range foundStickers {
		sticker.Used = sticker.Used + 1
		err := s.Stickers.SaveStickerFromPack(&sticker)
		if err != nil {
			log.Printf("Error: %v", err)
			return
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

func convertTgStickersToLocal(tgStickerPack tgbotapi.StickerSet) []Sticker {
	var stickers []Sticker
	for _, sticker := range tgStickerPack.Stickers {
		stickers = append(stickers, Sticker{
			FileId:       sticker.FileID,
			UniqueFileId: sticker.FileUniqueID,
			Text:         tgStickerPack.Title,
			Emoji:        sticker.Emoji,
			Pack:         sticker.SetName,
			AddedBy:      0,
			Private:      false,
		})
	}
	return stickers
}
