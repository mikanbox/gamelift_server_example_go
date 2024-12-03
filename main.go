package gamelift_server_example_go

import (
	"flag"
	"github.com/mikanbox/gamelift_server_example_go/modules"
)

func main() {
	webSocketURLArg := flag.String("webSocketURL", "wss://ap-northeast-1.api.amazongamelift.com", "WebSocket URL for sync gamelift status")
	hostIDArg := flag.String("hostID", "", "Compute name with RegisterCompute API")
	fleetIDArg := flag.String("fleetID", "", "Fleet ID")
	authTokenArg := flag.String("authToken", "", "Auth Token")
	portArg := flag.String("port", "8080", "Port")
	fleetTypeArg := flag.String("fleetType", "MANAGED", "Fleet type")

	flag.Parse()

	modules.AddExampleHTTPServer(
		*webSocketURLArg,
		*hostIDArg,
		*fleetIDArg,
		*authTokenArg,
		*portArg,
		*fleetTypeArg,
	)
}
