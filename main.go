package main

import (
	"os"
	"os/signal"
	"syscall"

	"tghomebot/qbittorrent"
	"tghomebot/storage"
	"tghomebot/tgbot"
	"tghomebot/utils"

	"github.com/sirupsen/logrus"
)

//Config ...
type Config struct {
	TelegramToken string `env:"TG_TOKEN"`
	SystemPort    int    `env:"SYS_PORT"`

	QBittorrentURL      string `env:"QBIT_URL"`
	QBittorrentLogin    string `env:"QBIT_LOGIN"`
	QBittorrentPassword string `env:"QBIT_PASS"`
}

func main() {
	cfg := &Config{}
	logger := logrus.New()
	logger.Out = os.Stdout

	if err := utils.ParseConfig(cfg); err != nil {
		logger.Panic(utils.Wrap("parse config:", err))
	}

	go utils.StartSystemServer(logger, cfg.SystemPort)

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	api := qbittorrent.NewAPIClient(cfg.QBittorrentURL, cfg.QBittorrentLogin, cfg.QBittorrentPassword)
	storage := storage.NewStorage("/data/torrent_bot.json", logger)
	bot, err := tgbot.NewBot(cfg.TelegramToken, storage, api, logger)
	if err != nil {
		logger.Panic(err)
	}
	if err = bot.Start(); err != nil {
		logger.Panic(err)
	}
	<-term
	logger.Infof("shutdown")
}
