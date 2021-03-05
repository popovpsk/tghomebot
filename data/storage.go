package data

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"tghomebot/api"
	"tghomebot/utils"

	"github.com/mailru/easyjson"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	filePath           string
	data               *api.Data
	dataRwm            sync.RWMutex
	fileM              sync.Mutex
	logger             *logrus.Logger
	isDataSourceExists bool
}

func NewStorage(filePath string, logger *logrus.Logger) *Storage {
	s := &Storage{
		filePath:           filePath,
		logger:             logger,
		isDataSourceExists: true,
	}
	err := s.loadData()
	if err != nil {
		s.logger.Error(fmt.Errorf("storage load data: %w", err))
		s.isDataSourceExists = false
	}
	return s
}

func (s *Storage) AddChatIfNotExists(chatID int64) {
	_, ok := s.data.Chats[chatID]
	if ok {
		return
	}
	s.dataRwm.Lock()
	s.data.Chats[chatID] = struct{}{}
	s.dataRwm.Unlock()
	if s.isDataSourceExists {
		go s.saveData()
	}
}

func (s *Storage) GetChats() []int64 {
	i := 0
	s.dataRwm.RLock()
	defer s.dataRwm.RUnlock()
	result := make([]int64, len(s.data.Chats))
	for k := range s.data.Chats {
		result[i] = k
		i++
	}
	return result
}

func (s *Storage) loadData() error {
	s.data = &api.Data{Chats: map[int64]struct{}{}}

	if err := os.MkdirAll(s.filePath, os.ModePerm); err != nil {
		return utils.WrapError("mkdir", err)
	}
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		f, _ := os.Create(s.filePath)
		w := bufio.NewWriter(f)
		s.data = &api.Data{Chats: map[int64]struct{}{}}
		_, err = easyjson.MarshalToWriter(s.data, w)
		if err != nil {
			return utils.WrapError("marshal to writer", err)
		}
		if err = w.Flush(); err != nil {
			return utils.WrapError("writer flush", err)
		}
		if err = f.Close(); err != nil {
			return utils.WrapError("closing file", err)
		}
	} else if err != nil {
		return utils.WrapError("file stat", err)
	} else {
		f, err := os.OpenFile(s.filePath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return utils.WrapError("open file", err)
		}
		rdr := bufio.NewReader(f)
		err = easyjson.UnmarshalFromReader(rdr, s.data)
		if err != nil {
			return utils.WrapError("unmarshal json from file", err)
		}
		err = f.Close()
		if err != nil {
			return utils.WrapError("closing file", err)
		}
	}
	return nil
}

func (s *Storage) saveData() {
	s.fileM.Lock()
	defer s.fileM.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_RDWR, os.ModePerm)
	if err != nil {
		err = utils.WrapError("storage save data: open file", err)
		s.logger.Error(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Error(utils.WrapError("saveData: closing file", err))
		}
	}()
	err = f.Truncate(0)
	if err != nil {
		err = utils.WrapError("storage save data: truncate file", err)
		s.logger.Error(err)
		return
	}
	w := bufio.NewWriter(f)
	if _, err = easyjson.MarshalToWriter(s.data, w); err != nil {
		logrus.Error(utils.WrapError("saveData: MarshalToWriter", err))
	}
	if err = w.Flush(); err != nil {
		logrus.Error(utils.WrapError("saveData: flush writer", err))
	}
}
