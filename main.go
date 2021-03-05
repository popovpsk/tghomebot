package main

import (
	"os"
	"os/signal"
	"syscall"

	"tghomebot/data"
	"tghomebot/qbittorrent"
	"tghomebot/tgbot"

	"github.com/sirupsen/logrus"
)

const (
	token          = "token"
	qBittorrentUrl = "http://192.168.1.49:8082"
)

func main() {
	logger := logrus.New()
	logger.Out = os.Stdout

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	api := qbittorrent.NewApi(qBittorrentUrl)
	storage := data.NewStorage(os.Getenv("HOME")+"/.tgbot/torrent_bot.json", logger)
	bot, err := tgbot.NewBot(token, storage, api, logger)
	if err != nil {
		logger.Fatal(err)
	}
	if err = bot.Start(); err != nil {
		logger.Fatal(err)
	}
	<-term
	logger.Infof("shutdown")
}
