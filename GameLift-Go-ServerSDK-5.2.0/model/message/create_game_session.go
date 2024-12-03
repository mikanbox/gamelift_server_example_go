/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

import "aws/amazon-gamelift-go-sdk/model"

// CreateGameSessionMessage - Message from GameLift initializing GameSession Creation
type CreateGameSessionMessage struct {
	Message
	MaximumPlayerSessionCount int               `json:"MaximumPlayerSessionCount"`
	Port                      int               `json:"Port"`
	IPAddress                 string            `json:"IpAddress"`
	GameSessionID             string            `json:"GameSessionId"`
	GameSessionName           string            `json:"GameSessionName"`
	GameSessionData           string            `json:"GameSessionData"`
	MatchmakerData            string            `json:"MatchmakerData"`
	DNSName                   string            `json:"DnsName"`
	GameProperties            map[string]string `json:"GameProperties"`
}

func NewGameSession(gameSession *CreateGameSessionMessage) *model.GameSession {
	if gameSession == nil {
		return nil
	}
	return &model.GameSession{
		GameSessionID:             gameSession.GameSessionID,
		GameSessionData:           gameSession.GameSessionData,
		Name:                      gameSession.GameSessionName,
		MatchmakerData:            gameSession.MatchmakerData,
		MaximumPlayerSessionCount: gameSession.MaximumPlayerSessionCount,
		IPAddress:                 gameSession.IPAddress,
		Port:                      gameSession.Port,
		DNSName:                   gameSession.DNSName,
		GameProperties:            gameSession.GameProperties,
	}
}
