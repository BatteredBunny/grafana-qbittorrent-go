package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func AddCookies(request *http.Request, cookies []*http.Cookie) {
	for _, v := range cookies {
		request.AddCookie(v)
	}
}

type Torrent struct {
	Hash string `json:"hash"`
}

type PeerRequest struct {
	Peers map[string]struct {
		Ip string
	} `json:"peers"`
}

func (qBittorrent *qBittorrent) login() (err error) {
	log.Println("Logging into qbittorrent")
	loginRequest, err := http.NewRequest("GET", qBittorrent.Address+"/api/v2/auth/login?username="+qBittorrent.Username+"&password="+qBittorrent.Password, nil)
	if err != nil {
		return
	}

	AddCookies(loginRequest, qBittorrent.Cookies)

	res, err := qBittorrent.Client.Do(loginRequest)
	if err != nil {
		return
	}

	qBittorrent.Cookies = res.Cookies()
	return
}

func (qBittorrent *qBittorrent) peers(torrent *Torrent) (err error) {
	peersRequest, err := http.NewRequest("GET", qBittorrent.Address+"/api/v2/sync/torrentPeers?hash="+torrent.Hash, nil)
	if err != nil {
		return
	}

	AddCookies(peersRequest, qBittorrent.Cookies)

	res, err := qBittorrent.Client.Do(peersRequest)
	if err != nil {
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	var peer PeerRequest
	if err = json.Unmarshal(data, &peer); err != nil {
		return
	}

	for _, element := range peer.Peers {
		qBittorrent.Ips = append(qBittorrent.Ips, element.Ip)
	}

	return
}

func (qBittorrent *qBittorrent) info() (torrents []Torrent, err error) {
	log.Println("Querying torrents info from qbittorrent")
	infoRequest, err := http.NewRequest("GET", qBittorrent.Address+"/api/v2/torrents/info", nil)
	if err != nil {
		return
	}

	AddCookies(infoRequest, qBittorrent.Cookies)

	res, err := qBittorrent.Client.Do(infoRequest)
	if err != nil {
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &torrents)

	return
}
