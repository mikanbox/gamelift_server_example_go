/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal

import (
	"encoding/json"
	"net/url"
	"time"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/server/log"
)

// IGameLiftManager - managing a single WebSocketClient, enabling connection and communication with GameLift.
//
//go:generate mockgen -destination ./mock/manager.go -package=mock . IGameLiftManager
type IGameLiftManager interface {
	Connect(websocketURL, processID, hostID, fleetID, authToken string, sigV4QueryParameters map[string]string) error
	Disconnect() error
	SendMessage(msg any) error
	HandleRequest(request MessageGetter, response any, timeout time.Duration) error
}

type gameLiftManager struct {
	handlers IGameLiftMessageHandler
	client   IWebSocketClient
	lg       log.ILogger
}

func GetGameLiftManager(
	handlers IGameLiftMessageHandler,
	client IWebSocketClient,
	lg log.ILogger,
) IGameLiftManager {
	gamelift := &gameLiftManager{
		handlers: handlers,
		client:   client,
		lg:       lg,
	}
	return gamelift
}

func (manager *gameLiftManager) Connect(websocketURL, processID, hostID, fleetID, authToken string, sigV4QueryParameters map[string]string) error {
	manager.lg.Debugf("Connecting to GameLift websocket server. Websocket URL: %s, processId: %s, hostId: %s, fleetId: %s", websocketURL, processID, hostID, fleetID)
	connectURL, err := url.Parse(websocketURL)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Add(common.PidKey, processID)
	params.Add(common.SdkVersionKey, common.SdkVersion)
	params.Add(common.SdkLanguageKey, common.SdkLanguage)
	params.Add(common.ComputeIDKey, hostID)
	params.Add(common.FleetIDKey, fleetID)
	if authToken != "" {
		params.Add(common.AuthTokenKey, authToken)
	} else if sigV4QueryParameters != nil {
		for key, value := range sigV4QueryParameters {
			params.Add(key, value)
		}
	}
	connectURL.RawQuery = params.Encode()

	if err := manager.client.Connect(connectURL); err != nil {
		return err
	}

	manager.client.AddHandler(message.CreateGameSession, manager.onStartGameSession)
	manager.client.AddHandler(message.UpdateGameSession, manager.onUpdateGameSession)
	manager.client.AddHandler(message.RefreshConnection, manager.onRefreshConnection)
	manager.client.AddHandler(message.TerminateProcess, manager.onTerminateProcess)

	return nil
}

func (manager *gameLiftManager) Disconnect() error {
	if err := manager.client.Close(); err != nil {
		return err
	}
	return nil
}

func (manager *gameLiftManager) SendMessage(msg any) error {
	return manager.client.SendMessage(msg)
}

// HandleRequest - send a request wait the response and parse it
// return error if timeout was expired or send request failed or can not parse answer.
func (manager *gameLiftManager) HandleRequest(request MessageGetter, response any, timeout time.Duration) error {
	respData := make(chan common.Outcome, 1)
	if err := manager.client.SendRequest(request, respData); err != nil {
		return err
	}

	expire := time.After(timeout)
	select {
	case <-expire:
		manager.client.CancelRequest(request.GetMessage().RequestID)
		manager.lg.Errorf("Response not received within time limit for request: %s", request.GetMessage().RequestID)
		return common.NewGameLiftError(common.ServiceCallFailed, "", "")
	case resultData := <-respData:
		if resultData.Error != nil {
			return resultData.Error
		}

		if err := json.Unmarshal(resultData.Data, response); err != nil {
			manager.lg.Errorf("Failed when try parse response data: %s", err.Error())
			return common.NewGameLiftError(common.InternalServiceException, "", "")
		}
		return nil
	}
}

func (manager *gameLiftManager) onStartGameSession(data []byte) {
	var gameSession message.CreateGameSessionMessage
	if err := json.Unmarshal(data, &gameSession); err != nil {
		manager.lg.Warnf("Failed when try parse start game session message: %s", err.Error())
		return
	}
	manager.handlers.OnStartGameSession(message.NewGameSession(&gameSession))
}

func (manager *gameLiftManager) onUpdateGameSession(data []byte) {
	var updateGameSession message.UpdateGameSessionMessage
	if err := json.Unmarshal(data, &updateGameSession); err != nil {
		manager.lg.Warnf("Failed when try parse update game session message: %s", err.Error())
		return
	}
	manager.handlers.OnUpdateGameSession(
		&updateGameSession.GameSession,
		updateGameSession.UpdateReason,
		updateGameSession.BackfillTicketID,
	)
}

func (manager *gameLiftManager) onTerminateProcess(data []byte) {
	var terminateProcess message.TerminateProcessMessage
	if err := json.Unmarshal(data, &terminateProcess); err != nil {
		manager.lg.Warnf("Failed when try parse terminate process message: %s", err.Error())
		return
	}
	manager.handlers.OnTerminateProcess(terminateProcess.TerminationTime)
}

func (manager *gameLiftManager) onRefreshConnection(data []byte) {
	var refreshConnection message.RefreshConnectionMessage
	if err := json.Unmarshal(data, &refreshConnection); err != nil {
		manager.lg.Warnf("Failed when try parse refresh connection message: %s", err.Error())
		return
	}
	manager.handlers.OnRefreshConnection(refreshConnection.RefreshConnectionEndpoint, refreshConnection.AuthToken)
}
