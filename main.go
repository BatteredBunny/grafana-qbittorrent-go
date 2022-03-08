package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
	"github.com/mmcloughlin/geohash"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Peer struct {
	Ip      string
	Geohash string
}

type GeoLocateResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Config struct {
	DBConnectionString string `toml:"db_connection_string"`
	Torrent            struct {
		Address  string `toml:"address"`
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"qbittorrent"`
}

func main() {
	var configLocation string
	flag.StringVar(&configLocation, "c", "config.toml", "Location of config file")
	flag.Parse()

	config, err := initializeConfig(configLocation)
	if err != nil {
		log.Fatal(err)
	}

	qBittorrent, err := initializeQbittorrent(&config)
	if err != nil {
		log.Fatal(err)
	}

	db, err := config.connectDB()
	if err != nil {
		log.Fatal(err)
	}

	for {
		torrents, err := qBittorrent.info()
		if err != nil {
			log.Println(err)
			continue
		}

		for _, v := range torrents {
			if err = qBittorrent.peers(&v); err != nil {
				log.Println(err)
				continue
			}
		}

		qBittorrent.sendToDB(db)
	}
}

func (qBittorrent *qBittorrent) sendToDB(db *sql.DB) {
	for _, v := range qBittorrent.Ips {
		geoHash, err := geoLocate(&http.Client{}, v)
		if err != nil {
			continue
		}

		err = insertOrUpdate(db, Peer{
			Ip:      v,
			Geohash: geoHash,
		})

		if err != nil {
			log.Fatal(err)
		}
	}

	qBittorrent.Ips = []string{}
}

func insertOrUpdate(db *sql.DB, peer Peer) (err error) {
	log.Println("Logging", peer.Ip)
	_, err = db.Exec("INSERT INTO public.peers (ip, geohash, last_saw, first_saw) VALUES($1::inet, $2, $3, $3) ON CONFLICT (ip) DO UPDATE SET last_saw=$3;", peer.Ip, peer.Geohash, time.Now().Unix())
	return
}

func geoLocate(client *http.Client, ip string) (geoHash string, err error) {
	request, err := http.NewRequest("GET", "https://geolocation-db.com/jsonp/"+ip, nil)
	if err != nil {
		return
	}

	res, err := client.Do(request)
	if err != nil {
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	data = bytes.TrimPrefix(data, []byte("callback("))
	data = bytes.TrimSuffix(data, []byte(")"))

	var coordinates GeoLocateResponse
	if err = json.Unmarshal(data, &coordinates); err != nil {
		return
	}

	geoHash = geohash.Encode(coordinates.Latitude, coordinates.Longitude)

	return
}

func prepareDb(db *sql.DB) (err error) {
	_, err = db.Exec("create table if not exists peers(ip inet not null constraint peers_pk primary key, geohash text not null, last_saw int not null, first_saw int  not null); create unique index if not exists peers_ip_uindex on peers (ip);")
	return
}

func (config Config) connectDB() (db *sql.DB, err error) {
	log.Println("Connecting to DB")
	db, err = sql.Open("postgres", config.DBConnectionString)
	if err != nil {
		return
	}

	err = prepareDb(db)

	return
}

func initializeConfig(configLocation string) (config Config, err error) {
	log.Println("Starting to read config")
	rawConfig, err := os.ReadFile(configLocation)
	if err != nil {
		return
	}

	config = Config{}

	if err = toml.Unmarshal(rawConfig, &config); err != nil {
		return
	}

	if config.DBConnectionString == "" {
		log.Fatal("No postgres connection string provided")
	} else if config.Torrent.Address == "" {
		log.Fatal("No qBittorrent address provided")
	} else if config.Torrent.Password == "" {
		log.Fatal("No qBittorrent password provided")
	} else if config.Torrent.Username == "" {
		log.Fatal("No qBittorrent username provided")
	}

	return
}
