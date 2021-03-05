package api

//go:generate easyjson

//easyjson:json
type (
	//Torrent ...
	Torrent struct {
		Name  string `json:"name"`
		State string `json:"state"`
		Hash  string `json:"hash"`
	}
	//Torrents ...
	Torrents []Torrent

	//Data ...
	Data struct {
		Chats map[int64]struct{}
	}
)
