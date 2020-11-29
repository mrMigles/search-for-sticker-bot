package main

import (
	"github.com/go-bongo/bongo"
	"gopkg.in/mgo.v2/bson"
	"log"
)

var mongoConnection = GetEnv("MONGO_CONNECTION", "")
var databaseName = GetEnv("DATABASE_NAME", "voice-mail")

type Sticker struct {
	bongo.DocumentBase `bson:",inline"`
	FileId             string `json:"-,"`
	Text               string `json:"-,"`
	Emoji              string `json:"-,"`
	Pack               string `json:"-,"`
	AddedBy            int    `json:"-,"`
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

func (m StickerResource) SaveSticker(user *Sticker) error {
	return m.connection.Collection("stickers").Save(user)
}

func (m StickerResource) FindStickers(text string) []Sticker {
	results := m.connection.Collection("stickers").Find(bson.M{"$text": map[string]string{"$search": text}})
	var stickers []Sticker
	sticker := &Sticker{}
	for results.Next(sticker) {
		stickers = append(stickers, *sticker)
	}
	return stickers
}
