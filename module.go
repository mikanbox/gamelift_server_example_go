package gamelift_server_example_go

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// GameLiftConfig 構造体は、GameLift サーバーの設定を保持
type GameLiftConfig struct {
	WebSocketURL string
	ProcessID    string
	HostID       string
	FleetID      string
	AuthToken    string
	Port         int
	FleetType    string
}

// AddExampleHTTPServer は、HTTP サーバーを追加する関数

func AddExampleHTTPServer(webSocketURLArg, hostIDArg, fleetIDArg, authTokenArg, portArg, fleetTypeArg string) {
	// fleetTypeArg が "MANAGED" であれば、MANAGED フリートを作成する。多くの変数は環境変数で上書きされる
	if fleetTypeArg == "MANAGED" {
		addExampleHTTPServer("", "", "", "", portArg, "MANAGED")
	} else {
		addExampleHTTPServer(webSocketURLArg, hostIDArg, fleetIDArg, authTokenArg, portArg, "ANYWHERE")
	}
}

// addExampleHTTPServer は、HTTP サーバーを初期化して起動
func addExampleHTTPServer(webSocketURLArg, hostIDArg, fleetIDArg, authTokenArg, portArg, fleetTypeArg string) {
	processUUID, _ := uuid.NewUUID()
	config := GameLiftConfig{
		WebSocketURL: webSocketURLArg,
		ProcessID:    "processid" + processUUID.String(),
		HostID:       hostIDArg,
		FleetID:      fleetIDArg,
		AuthToken:    authTokenArg,
		Port:         func() int { port, _ := strconv.Atoi(portArg); return port }(),
		FleetType:    fleetTypeArg,
	}
	flag.Parse()

	logpath := setupLogging(config)
	defer func() {
		if file, err := os.OpenFile(logpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			defer file.Close()
		}
	}()

	shutdownChan := make(chan struct{})
	setup(config.FleetType, config.WebSocketURL, config.ProcessID, config.HostID, config.FleetID, config.AuthToken, config.Port, logpath, shutdownChan)
	log.Print("[CustomDebug]:end execScripts num goroutine in main: ", runtime.NumGoroutine())

	http.HandleFunc("/", homePage)
	http.HandleFunc("/gamesessionid", getGameSessionId)
	http.HandleFunc("/quit", quit(shutdownChan))
	http.HandleFunc("/desc", describePlayers)
	http.HandleFunc("/accept", acceptPlayerSession)
	http.HandleFunc("/removeplayer", removePlayerSession)
	http.HandleFunc("/backfill", backfillRequest)
	http.HandleFunc("/maker", showMatchMaker)

	srv := &http.Server{Addr: ":" + strconv.Itoa(config.Port)}

	go func() {
		<-shutdownChan
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
	}()

	log.Fatal(srv.ListenAndServe())
}

// setupLogging は、ログの設定を行う関数
func setupLogging(config GameLiftConfig) string {
	logpath := "./gameserver.log"
	if config.FleetType == "MANAGED" {
		logpath = "/local/game/gameserver" + time.Now().String() + config.ProcessID + ".log"
	} else {
		logpath = "./gameserver.log"
	}
	file, err := os.OpenFile(logpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	return logpath
}
