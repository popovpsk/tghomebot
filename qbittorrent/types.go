package qbittorrent

type Torrent struct {
	Name          string `json:"name"`
	ExternalState string `json:"state"`
	Hash          string `json:"hash"`
}

func (t *Torrent) State() TorrentState {
	// Torrent State
	const (
		queuedUPState    = "queuedUP"
		checkingDLState  = "checkingDL"
		downloadingState = "downloading"
		uploadingState   = "uploading"
	)

	switch t.ExternalState {
	case queuedUPState, uploadingState:
		return Uploading
	case checkingDLState, downloadingState:
		return Downloading
	default:
		return Undefined
	}
}

type TorrentState int16

const (
	Downloading TorrentState = iota + 1
	Uploading
	Undefined
)
