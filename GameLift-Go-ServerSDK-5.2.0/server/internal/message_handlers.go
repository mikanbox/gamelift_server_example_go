/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal

import (
	"aws/amazon-gamelift-go-sdk/model"
)

// IGameLiftMessageHandler - async messages handlers from GameLift server or APIG.
//
//go:generate mockgen -destination ../internal/mock/handlers.go -package=mock . IGameLiftMessageHandler
type IGameLiftMessageHandler interface {
	OnStartGameSession(gameSession *model.GameSession)
	OnUpdateGameSession(
		gameSession *model.GameSession,
		updateReason *model.UpdateReason,
		backfillTicketID string,
	)
	OnTerminateProcess(terminationTime int64)
	OnRefreshConnection(refreshConnectionEndpoint, authToken string)
}
