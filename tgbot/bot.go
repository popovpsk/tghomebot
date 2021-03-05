package tgbot

import (
	"fmt"
	"sync"
	"time"

	"tghomebot/data"
	"tghomebot/qbittorrent"
	"tghomebot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Bot struct {
	botApi      *tgbotapi.BotAPI
	data        *data.Storage
	api         *qbittorrent.Client
	token       string
	events      chan string
	downloading sync.Map
	logger      *logrus.Logger
}

func NewBot(token string, data *data.Storage, api *qbittorrent.Client, logger *logrus.Logger) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, utils.WrapError("new telegram bot", err)
	}
	logger.Infof("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		api:         api,
		data:        data,
		token:       token,
		botApi:      bot,
		downloading: sync.Map{},
		events:      make(chan string),
		logger:      logger,
	}, nil
}

func (b *Bot) Start() error {
	b.startWatching()
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updCh, err := b.botApi.GetUpdatesChan(updateConfig)
	if err != nil {
		return err
	}
	go func() {
		for event := range b.events {
			for _, chatID := range b.data.GetChats() {
				msg := tgbotapi.NewMessage(chatID, event)
				if _, err = b.botApi.Send(msg); err != nil {
					logrus.Error(utils.WrapError("bot send event message", err))
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
		err = utils.WrapError("message handling", err)
		b.logger.Error(err)
		msg := tgbotapi.NewMessage(chatID, err.Error())
		_, err = b.botApi.Send(msg)
		if err != nil {
			b.logger.Error(utils.WrapError("bot sending message", err))
		}
	}
}

func (b *Bot) handleFile(update tgbotapi.Update) error {
	fileID := update.Message.Document.FileID
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := b.botApi.GetFile(fileConfig)
	if err != nil {
		return utils.WrapError("get telegram file info", err)
	}

	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(file.Link(b.token))
	err = fasthttp.Do(req, resp)
	if err != nil {
		return utils.WrapError("download telegram file", err)
	}

	if err := b.api.SendFile(resp.Body()); err != nil {
		return utils.WrapError("api call error", err)
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
		logrus.Error(utils.WrapError("watch: get torrents info", err))
	}
	for _, t := range torrents {
		_, ok := b.downloading.Load(t.Hash)
		switch t.State {
		case qbittorrent.QueuedUPState, qbittorrent.UploadingState:
			if ok {
				b.downloading.Delete(t.Hash)
				go func() {
					b.events <- fmt.Sprintf("<--Загружен-->\n%s", t.Name)
				}()
			}
		case qbittorrent.DownloadingState, qbittorrent.CheckingDLState:
			if !ok {
				b.downloading.Store(t.Hash, t.Name)
				go func() {
					b.events <- fmt.Sprintf("<--Загружается-->\n%s", t.Name)
				}()
			}
		}
	}
}
