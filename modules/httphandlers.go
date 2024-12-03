package modules

import (
	"fmt"
	"net/http"
)

func quit(shutdownChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Quit ")
		fmt.Println("Quit ")
		processTerminate(shutdownChan)
	}
}

func getGameSessionId(w http.ResponseWriter, r *http.Request) {
	gameSessionID := getgamesesionId()
	fmt.Fprintf(w, "GameLift GameSession ID :"+gameSessionID)
	fmt.Println("GameLift GameSession ID :" + gameSessionID)
}

func describePlayers(w http.ResponseWriter, r *http.Request) {
	res := getgamesesionId()
	fmt.Fprintf(w, "GameLift Info \n "+res)
	fmt.Println(w, "GameLift Info \n "+res)
}

func removePlayerSession(w http.ResponseWriter, r *http.Request) {
	len := r.ContentLength
	body := make([]byte, len) // Content-Length と同じサイズの byte 配列を用意
	r.Body.Read(body)         // byte 配列にリクエストボディを読み込む
	playerid := string(body)

	removeplayersession(playerid)
	fmt.Fprintf(w, "GameLift Remove Player Session ID :"+playerid)
	fmt.Println(w, "GameLift Remove Player Session ID :"+playerid)
}

func acceptPlayerSession(w http.ResponseWriter, r *http.Request) {
	len := r.ContentLength
	body := make([]byte, len) // Content-Length と同じサイズの byte 配列を用意
	r.Body.Read(body)         // byte 配列にリクエストボディを読み込む
	playerid := string(body)

	acceptplayer(playerid)
	fmt.Fprintf(w, "GameLift Accept Player Session ID :"+playerid)
	fmt.Println(w, "GameLift Accept Player Session ID :"+playerid)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Response for user")
	fmt.Println("Request receive")
}

func backfillRequest(w http.ResponseWriter, r *http.Request) {
	backfill()
	fmt.Fprintf(w, "GameLift Backfill \n ")
	fmt.Println(w, "GameLift Backfill \n ")
}

func showMatchMaker(w http.ResponseWriter, r *http.Request) {
	res := resMatchMakerData()
	fmt.Fprintf(w, "GameLift Backfill \n "+res)
	fmt.Println(w, "GameLift Backfill \n "+res)
}
