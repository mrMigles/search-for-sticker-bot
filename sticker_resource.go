package main

import (
	"github.com/go-bongo/bongo"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net"
)

var mongoConnection = GetEnv("MONGO_CONNECTION", "")
var databaseName = GetEnv("DATABASE_NAME", "voice-mail")

type Sticker struct {
	bongo.DocumentBase `bson:",inline"`
	FileId             string `json:"-,"`
	UniqueFileId       string `json:"-,"`
	Text               string `json:"-,"`
	Emoji              string `json:"-,"`
	Pack               string `json:"-,"`
	AddedBy            int    `json:"-,"`
	Private            bool   `json:"-,"`
}

type User struct {
	bongo.DocumentBase `bson:",inline"`
	UserId             int  `json:"-,"`
	Private            bool `json:"-,"`
}

type StickerResource struct {
	connection *bongo.Connection
}

func NewStickerResource() *StickerResource {
	config := &bongo.Config{
		ConnectionString: mongoConnection,
		Database:         databaseName,
	}
	connection, err := bongo.Connect(config)
	if err != nil {
		log.Fatal(err)
	}
	connection.Session.SetPoolLimit(50)
	return &StickerResource{connection: connection}
}

func (m StickerResource) SaveSticker(sticker *Sticker) error {
	return m.connection.Collection("stickers").Save(sticker)
}

func (m StickerResource) SaveUser(user *User) error {
	return m.connection.Collection("sticker_users").Save(user)
}

func (m StickerResource) ChangeUserType(user *User) error {
	err := m.connection.Collection("sticker_users").Save(user)
	if err != nil {
		return err
	}
	userStickers := m.FindStickersByUser(*user)
	for _, sticker := range userStickers {
		sticker.Private = user.Private
		err = m.SaveSticker(&sticker)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m StickerResource) FindUser(userId int) (*User, error) {
	user := &User{}
	err := m.connection.Collection("sticker_users").FindOne(bson.M{"userid": userId}, user)

	if err != nil {
		if _, ok := err.(*net.OpError); ok {
			return nil, err
		}
		log.Printf("User %s not found", userId)
		return nil, nil
	}
	return user, nil
}

func (m StickerResource) FindPublicStickersByText(text string) []Sticker {
	results := m.connection.Collection("stickers").Find(bson.M{"$and": []map[string]interface{}{{"$text": map[string]string{"$search": text}}, {"private": false}}})
	var stickers []Sticker
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}
	return stickers
}

func (m StickerResource) FindPublicStickersByFileId(fileId string) []Sticker {
	results := m.connection.Collection("stickers").Find(bson.M{"$and": []map[string]interface{}{{"uniquefileid": fileId}, {"private": false}}})
	var stickers []Sticker
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}
	return stickers
}

func (m StickerResource) FindPublicStickersByFileIdAndUser(fileId string, user User) []Sticker {
	results := m.connection.Collection("stickers").Find(bson.M{"$and": []map[string]interface{}{{"uniquefileid": fileId}, {"addedby": user.UserId}}})
	var stickers []Sticker
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}
	return stickers
}

func (m StickerResource) FindStickersByTextAndUser(text string, user User) []Sticker {
	var stickers []Sticker

	results := m.connection.Collection("stickers").Find(bson.M{"$and": []map[string]interface{}{{"$text": map[string]string{"$search": text}}, {"addedby": user.UserId}}})
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}

	if !user.Private {
		allStickers := m.FindPublicStickersByText(text)
		for _, sticker := range allStickers {
			if !containsSticker(stickers, sticker) {
				stickers = append(stickers, sticker)
			}
		}
	}
	return stickers
}

func (m StickerResource) FindStickersByUser(user User) []Sticker {
	results := m.connection.Collection("stickers").Find(bson.M{"addedby": user.UserId})
	var stickers []Sticker
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}
	return stickers
}

func containsSticker(stickers []Sticker, sticker Sticker) bool {
	for _, stick := range stickers {
		if sticker.UniqueFileId == stick.UniqueFileId {
			return true
		}
	}
	return false
}
