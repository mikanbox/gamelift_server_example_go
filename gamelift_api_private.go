package gamelift_server_example_go

import (
	"encoding/json"
	"fmt"
	"github.com/mikanbox/gamelift_server_example_go/GameLift-Go-ServerSDK-5.2.0/model"
	"github.com/mikanbox/gamelift_server_example_go/GameLift-Go-ServerSDK-5.2.0/model/request"
	"github.com/mikanbox/gamelift_server_example_go/GameLift-Go-ServerSDK-5.2.0/server"
	"log"
	"os"
	"runtime"
)

type gameProcess struct {
	Port int
	Logs server.LogParameters
}

// GameSession開始を受信するコールバック
func (g gameProcess) OnStartGameSession(myGameSession model.GameSession) {
	fmt.Println("Callback: OnStartGameSession")
	err := server.ActivateGameSession()
	if err != nil {
		log.Fatal(err.Error())
	}

	jsonout, err := json.Marshal(myGameSession)

	if err == nil {
		fmt.Println("GameLift Info \n " + string(jsonout))
	}
	globalgamesession = myGameSession
}

var globalgamesession model.GameSession

// GameSession更新を受信するコールバック
func (g gameProcess) OnUpdateGameSession(myGameSession model.UpdateGameSession) {
	fmt.Println("Callback: OnUpdateGameSession")
	jsonout, err := json.Marshal(myGameSession)

	if err == nil {
		fmt.Println("GameLift Info \n " + string(jsonout))
	}
	globalgamesession = myGameSession.GameSession
}

// プロセス終了を受信するコールバック
func (g gameProcess) OnProcessTerminate(shutdownChan chan struct{}) {
	fmt.Println("Callback: OnProcessTerminate")
	err := server.ProcessEnding()
	if err != nil {
		close(shutdownChan) // Signal to shutdown HTTP server
		os.Exit(0)
	}
	defer server.Destroy()

	close(shutdownChan) // Signal to shutdown HTTP server

	os.Exit(0)
}

func processTerminate(shutdownChan chan struct{}) {
	fmt.Println("Callback: OnProcessTerminate")
	process.OnProcessTerminate(shutdownChan)
}

// GameLift からヘルスチェック受信するコールバック
func (g gameProcess) OnHealthCheck() bool {
	fmt.Println("Callback: OnHealthCheck")
	return true
}

func describePlayerSessions() string {
	describePlayerSessionsRequest := request.NewDescribePlayerSessions()
	describePlayerSessionsRequest.GameSessionID, _ = server.GetGameSessionID() // get ID for the current game session
	describePlayerSessionsRequest.Limit = 10

	res, err := server.DescribePlayerSessions(describePlayerSessionsRequest)
	if err != nil {
		log.Fatal(err.Error())
	}

	jsonout, err := json.Marshal(res)

	return string(jsonout)
}

func getgamesesionId() string {
	gameSessionID, err := server.GetGameSessionID()
	if err != nil {
		log.Fatal(err.Error())
	}
	return gameSessionID
}

func removeplayersession(playerid string) {
	err := server.RemovePlayerSession(playerid)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func acceptplayer(playerid string) {
	err := server.AcceptPlayerSession(playerid)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func resMatchMakerData() string {
	jsonout, err := json.Marshal(globalgamesession.MatchmakerData)
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(jsonout)
}

func backfill() {
	// form the request
	var matchMaker model.MatchmakerData
	if err := matchMaker.UnmarshalJSON([]byte(globalgamesession.MatchmakerData)); err != nil {
		return
	}

	players := matchMaker.Players
	players = players[:0+copy(players[0:], players[0+1:])]

	startBackfillRequest := request.NewStartMatchBackfill(
		globalgamesession.GameSessionID,
		"arn:aws:gamelift:ap-northeast-1:859690264691:matchmakingconfiguration/backfill",
		players,
	)

	jsonout, err := json.Marshal(globalgamesession.MatchmakerData)
	fmt.Println("players" + string(jsonout))
	fmt.Println("MatchmakerData " + globalgamesession.MatchmakerData)

	res, err := server.StartMatchBackfill(startBackfillRequest)
	if err != nil {
		log.Fatal(err.Error())
	} else {
		fmt.Println("Start Backfill " + res.TicketID)
	}
}

var process = gameProcess{}

func setup(fleettype string, websocketurl string, processid string, hostid string, fleetid string, authtoken string, port int, logpath string, shutdownChan chan struct{}) {

	var param server.ServerParameters
	if fleettype == "ANYWHERE" {
		fmt.Println("FleetTYPE: ANYWHERE")
		param = server.ServerParameters{
			WebSocketURL: websocketurl,
			ProcessID:    processid,
			HostID:       hostid,
			FleetID:      fleetid,
			AuthToken:    authtoken,
		}
	} else {
		fmt.Println("FleetTYPE: MAANGED")
		param = server.ServerParameters{}
	}

	log.Print("[CustomDebug]:FleetTYPE: " + fleettype)
	log.Print("[CustomDebug]:Invoke initSDK")
	err := server.InitSDK(param)
	if err != nil {
		log.Fatal(err.Error())
	}

	process := gameProcess{
		Port: port,
	}

	log.Print("[CustomDebug]:Invoke processReady")
	err = server.ProcessReady(server.ProcessParameters{
		OnStartGameSession:  process.OnStartGameSession,
		OnProcessTerminate:  func() { process.OnProcessTerminate(shutdownChan) },
		OnUpdateGameSession: process.OnUpdateGameSession,
		OnHealthCheck:       process.OnHealthCheck,
		Port:                process.Port,
		LogParameters: server.LogParameters{ // logging and error example
			LogPaths: []string{logpath},
		},
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	server.UpdatePlayerSessionCreationPolicy(model.AcceptAll)

	log.Print("[CustomDebug]:Function setup is ended")
	log.Print("[CustomDebug]:end execScripts num goroutine in gamelift_api: ", runtime.NumGoroutine())

}
