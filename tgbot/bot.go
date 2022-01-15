package tgbot

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"tghomebot/qbittorrent"
	"tghomebot/storage"
	"tghomebot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

//Bot ...
type Bot struct {
	botAPI      *tgbotapi.BotAPI
	data        *storage.Storage
	api         *qbittorrent.Client
	token       string
	events      chan string
	downloading *sync.Map
	logger      *logrus.Logger
}

//NewBot constructor
func NewBot(token string, data *storage.Storage, api *qbittorrent.Client, logger *logrus.Logger) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, utils.Wrap("new telegram bot", err)
	}
	logger.Infof("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		api:         api,
		data:        data,
		token:       token,
		botAPI:      bot,
		downloading: &sync.Map{},
		events:      make(chan string),
		logger:      logger,
	}, nil
}

//Start ...
func (b *Bot) Start() error {
	b.startWatching()
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updCh, err := b.botAPI.GetUpdatesChan(updateConfig)
	if err != nil {
		return err
	}
	go func() {
		for event := range b.events {
			for _, chatID := range b.data.GetChats() {
				msg := tgbotapi.NewMessage(chatID, event)
				if _, err = b.botAPI.Send(msg); err != nil {
					logrus.Error(utils.Wrap("bot send event message", err))
				}
			}
		}
	}()
	go func() {
		for update := range updCh {
			b.handleMessage(update)
		}
	}()
	return err
}

func (b *Bot) handleMessage(update tgbotapi.Update) {
	var err error
	chatID := update.Message.Chat.ID
	b.data.AddChatIfNotExists(chatID)

	if update.Message.Document != nil {
		err = b.handleFile(update)
	}
	if len(update.Message.Text) > 6 && update.Message.Text[:6] == "magnet" {
		err = b.api.SendMagnet([]byte(update.Message.Text))
	}

	if err != nil {
		err = utils.Wrap("message handling", err)
		b.logger.Error(err)
		msg := tgbotapi.NewMessage(chatID, err.Error())
		_, err = b.botAPI.Send(msg)
		if err != nil {
			b.logger.Error(utils.Wrap("bot sending message", err))
		}
	}
}

func (b *Bot) handleFile(update tgbotapi.Update) error {
	fileID := update.Message.Document.FileID
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := b.botAPI.GetFile(fileConfig)
	if err != nil {
		return utils.Wrap("get telegram file info", err)
	}

	resp, err := http.Get(file.Link(b.token))

	if err != nil {
		return utils.Wrap("download telegram file", err)
	}
	var body bytes.Buffer
	if resp.ContentLength > 0 {
		body.Grow(int(resp.ContentLength))
	}
	_, err = io.Copy(&body, resp.Body)
	if err != nil {
		return utils.Wrap("io.Copy body", err)
	}

	if err := b.api.SendFile(body.Bytes()); err != nil {
		return utils.Wrap("api call error", err)
	}
	return err
}

func (b *Bot) startWatching() {
	go func() {
		for range time.Tick(time.Second * 3) {
			b.watch()
		}
	}()
}

func (b *Bot) watch() {
	torrents, err := b.api.GetTorrentsInfo()
	if err != nil {
		logrus.Error(utils.Wrap("watch: get torrents info", err))
	}

	for _, t := range torrents {
		_, ok := b.downloading.Load(t.Hash)
		switch t.State() {
		case qbittorrent.Uploading:
			if ok {
				b.downloading.Delete(t.Hash)
				b.sendMessage(fmt.Sprintf("<--Загружен-->\n%s", t.Name))
			}
		case qbittorrent.Downloading:
			if !ok {
				b.downloading.Store(t.Hash, t.Name)
				b.sendMessage(fmt.Sprintf("<--Загружен-->\n%s", t.Name))
			}
		}
	}
}

func (b *Bot) sendMessage(message string) {
	select {
	case b.events <- message:
		return
	default:
		go func(msg string) {
			b.events <- msg
		}(message)
	}
}
