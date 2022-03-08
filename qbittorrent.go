package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
)

type qBittorrent struct {
	Ips          []string
	Client       *http.Client
	loginRequest *http.Request
	infoRequest  *http.Request
	peersRequest *http.Request
}

type Torrent struct {
	Hash string `json:"hash"`
}

type PeerRequest struct {
	Peers map[string]struct {
		Ip string
	} `json:"peers"`
}

func (qBittorrent *qBittorrent) login() (cookies []*http.Cookie, err error) {
	log.Println("Logging into qbittorrent")
	res, err := qBittorrent.Client.Do(qBittorrent.loginRequest)
	if err != nil {
		return
	}

	return res.Cookies(), nil
}

func (qBittorrent *qBittorrent) peers(torrent *Torrent) (err error) {
	qBittorrent.peersRequest.URL, err = url.Parse(qBittorrent.peersRequest.URL.Scheme + "://" + qBittorrent.peersRequest.URL.Host + "/api/v2/sync/torrentPeers?hash=" + torrent.Hash)
	if err != nil {
		return
	}

	res, err := qBittorrent.Client.Do(qBittorrent.peersRequest)
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
	res, err := qBittorrent.Client.Do(qBittorrent.infoRequest)
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

func initializeQbittorrent(config *Config) (QBittorrent *qBittorrent, err error) {
	QBittorrent = &qBittorrent{
		Client: &http.Client{},
	}

	QBittorrent.loginRequest, err = http.NewRequest("GET", config.Torrent.Address+"/api/v2/auth/login?username="+config.Torrent.Username+"&password="+config.Torrent.Password, nil)
	if err != nil {
		return
	}

	QBittorrent.infoRequest, err = http.NewRequest("GET", config.Torrent.Address+"/api/v2/torrents/info", nil)
	if err != nil {
		return
	}

	QBittorrent.peersRequest, err = http.NewRequest("GET", config.Torrent.Address, nil)
	if err != nil {
		return
	}

	cookies, err := QBittorrent.login()
	if err != nil {
		return
	}

	AddCookies(QBittorrent.loginRequest, cookies)
	AddCookies(QBittorrent.infoRequest, cookies)
	AddCookies(QBittorrent.peersRequest, cookies)

	return
}

func AddCookies(request *http.Request, cookies []*http.Cookie) {
	for _, v := range cookies {
		request.AddCookie(v)
	}
}
