module github.com/mikanbox/gamelift_server_example_go

go 1.21.0

replace aws/amazon-gamelift-go-sdk => ./GameLift-Go-ServerSDK-5.2.0

require (
	aws/amazon-gamelift-go-sdk v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
)

require (
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/sethvargo/go-retry v0.2.4 // indirect
	golang.org/x/net v0.20.0 // indirect
)
