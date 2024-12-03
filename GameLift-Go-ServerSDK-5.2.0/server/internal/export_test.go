/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal

import (
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
	"aws/amazon-gamelift-go-sdk/server/log"
)

type WebsocketClient = websocketClient

// Init expose access private init method for testing purposes
func (c *WebsocketClient) Init(transport transport.ITransport, logger log.ILogger) {
	c.init(transport, logger)
}

// RunReadHandler expose access private readHandler method for testing purposes
func (c *WebsocketClient) RunReadHandler(data []byte) {
	c.readHandler(data)
}
