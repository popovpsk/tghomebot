package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"tghomebot/api"
	"tghomebot/utils"

	"github.com/sirupsen/logrus"
)

//Storage ...
type Storage struct {
	filePath           string
	data               *api.Data
	dataRwm            sync.RWMutex
	fileM              sync.Mutex
	logger             *logrus.Logger
	isDataSourceExists bool
}

//NewStorage ...
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

//AddChatIfNotExists adds a chat to the list for events
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

//GetChats returning all chats for pub
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

	if err := os.MkdirAll(filepath.Dir(s.filePath), os.ModePerm); err != nil {
		return utils.Wrap("mkdir", err)
	}
	_, err := os.Stat(s.filePath)
	if os.IsNotExist(err) {
		f, _ := os.Create(s.filePath)
		s.data = &api.Data{Chats: map[int64]struct{}{}}
		b, _ := json.Marshal(s.data)
		_, err = f.Write(b)
		if err != nil {
			return utils.Wrap("file write", err)
		}
		if err = f.Close(); err != nil {
			return utils.Wrap("closing file", err)
		}
		return nil
	}

	if err != nil {
		return utils.Wrap("file stat", err)
	}

	f, err := os.OpenFile(s.filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return utils.Wrap("open file", err)
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(s.data)
	if err != nil {
		return utils.Wrap("unmarshal json from file", err)
	}
	err = f.Close()
	if err != nil {
		return utils.Wrap("closing file", err)
	}
	return nil
}

func (s *Storage) saveData() {
	s.fileM.Lock()
	defer s.fileM.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_RDWR, os.ModePerm)
	if err != nil {
		err = utils.Wrap("storage save data: open file", err)
		s.logger.Error(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Error(utils.Wrap("saveData: closing file", err))
		}
	}()
	err = f.Truncate(0)
	if err != nil {
		err = utils.Wrap("storage save data: truncate file", err)
		s.logger.Error(err)
		return
	}
	encoder := json.NewEncoder(f)
	err = encoder.Encode(s.data)
	if err != nil {
		logrus.Error(utils.Wrap("saveData: MarshalToWriter", err))
	}
}
