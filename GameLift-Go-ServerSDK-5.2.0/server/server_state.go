/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"aws/amazon-gamelift-go-sdk/server/internal/security"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model"
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/request"
	"aws/amazon-gamelift-go-sdk/model/result"
	"aws/amazon-gamelift-go-sdk/server/internal"
)

var localRnd *rand.Rand

const ActivateServerProcessRequestTimeoutInSeconds = time.Duration(6) * time.Second

func init() {
	//nolint:gosec // Use a weak random generator is enough in this case
	localRnd = rand.New(rand.NewSource(time.Now().Unix()))
}

type iGameLiftServerState interface {
	processReady(*ProcessParameters) error
	processEnding() error
	activateGameSession() error
	updatePlayerSessionCreationPolicy(*model.PlayerSessionCreationPolicy) error
	getGameSessionID() (string, error)
	getTerminationTime() (int64, error)
	acceptPlayerSession(playerSessionID string) error
	removePlayerSession(playerSessionID string) error
	describePlayerSessions(*request.DescribePlayerSessionsRequest) (result.DescribePlayerSessionsResult, error)
	startMatchBackfill(*request.StartMatchBackfillRequest) (result.StartMatchBackfillResult, error)
	stopMatchBackfill(*request.StopMatchBackfillRequest) error
	getComputeCertificate() (result.GetComputeCertificateResult, error)
	getFleetRoleCredentials(*request.GetFleetRoleCredentialsRequest) (result.GetFleetRoleCredentialsResult, error)
	destroy() error
}

type gameLiftServerState struct {
	wsGameLift internal.IGameLiftManager
	parameters *ProcessParameters

	processID string
	hostID    string
	fleetID   string

	gameSessionID   string
	terminationTime int64

	isReadyProcess common.AtomicBool
	onManagedEC2   bool

	fleetRoleResultCache map[string]result.GetFleetRoleCredentialsResult
	mtx                  sync.Mutex

	defaultJitterIntervalMs int64
	healthCheckInterval     time.Duration
	healthCheckTimeout      time.Duration
	serviceCallTimeout      time.Duration

	shutdown chan bool
}

func (state *gameLiftServerState) init(params *ServerParameters, wsGameLift internal.IGameLiftManager) error {
	if params == nil {
		return common.NewGameLiftError(common.GameLiftServerNotInitialized, "", "")
	}
	state.fleetRoleResultCache = make(map[string]result.GetFleetRoleCredentialsResult)
	state.processID = common.GetEnvStringOrDefault(common.EnvironmentKeyProcessID, params.ProcessID)
	state.hostID = common.GetEnvStringOrDefault(common.EnvironmentKeyHostID, params.HostID)
	state.fleetID = common.GetEnvStringOrDefault(common.EnvironmentKeyFleetID, params.FleetID)

	websocketUrl := common.GetEnvStringOrDefault(common.EnvironmentKeyWebsocketURL, params.WebSocketURL)
	computeType := common.GetEnvStringOrDefault(common.EnvironmentKeyComputeType, "")
	authToken := common.GetEnvStringOrDefault(common.EnvironmentKeyAuthToken, params.AuthToken)
	awsRegion := common.GetEnvStringOrDefault(common.EnvironmentKeyAwsRegion, params.AwsRegion)
	accessKey := common.GetEnvStringOrDefault(common.EnvironmentKeyAccessKey, params.AccessKey)
	secretKey := common.GetEnvStringOrDefault(common.EnvironmentKeySecretKey, params.SecretKey)
	sessionToken := common.GetEnvStringOrDefault(common.EnvironmentKeySessionToken, params.SessionToken)

	if common.GetEnvStringOrDefault(common.EnvironmentKeyProcessID, "") == common.AgentlessContainerProcessId {
		state.processID = uuid.New().String()
	}
	isContainerComputeType := computeType == common.ComputeTypeContainer
	authTokenPassed := authToken != ""
	sigV4ParametersPassed := awsRegion != "" && accessKey != "" && secretKey != ""
	if !authTokenPassed && !sigV4ParametersPassed && !isContainerComputeType {
		return common.NewGameLiftError(common.BadRequestException, "", "Either AuthToken or AwsRegion and AwsCredentials are required")
	}

	state.onManagedEC2 = true
	state.defaultJitterIntervalMs = common.GetEnvDurationOrDefault(
		common.HealthcheckMaxJitter,
		common.HealthcheckMaxJitterDefault,
		lg,
	).Milliseconds()
	state.healthCheckInterval = common.GetEnvDurationOrDefault(
		common.HealthcheckInterval,
		common.HealthcheckIntervalDefault,
		lg,
	)
	state.healthCheckTimeout = common.GetEnvDurationOrDefault(
		common.HealthcheckTimeout,
		common.HealthcheckTimeoutDefault,
		lg,
	)
	state.serviceCallTimeout = common.GetEnvDurationOrDefault(
		common.ServiceCallTimeout,
		common.ServiceCallTimeoutDefault,
		lg,
	)

	var sigV4QueryParameters map[string]string
	if !authTokenPassed {
		if isContainerComputeType {
			httpClient := &http.Client{}

			credentialFetcher, err := security.NewContainerCredentialsFetcher(httpClient)
			if err != nil {
				log.Fatalf("Failed to create Container Credentials Fetcher: %v", err)
				return err
			}
			awsCredentials, err := credentialFetcher.FetchContainerCredentials()
			if err != nil {
				log.Fatalf("Failed to fetch container credentials: %v", err)
				return err
			}
			accessKey = awsCredentials.AccessKey
			secretKey = awsCredentials.SecretKey
			sessionToken = awsCredentials.SessionToken

			containerMetadataFetcher, err := security.NewContainerMetadataFetcher(httpClient)
			if err != nil {
				log.Fatalf("Failed to create Container Metadata Fetcher: %v", err)
				return err
			}
			containerTaskMetadata, err := containerMetadataFetcher.FetchContainerTaskMetadata()
			if err != nil {
				log.Fatalf("Failed to fetch container task metadata: %v", err)
				return err
			}

			state.hostID = containerTaskMetadata.TaskId
		}
		sigV4QueryParameters = getSigV4QueryParameters(awsRegion, accessKey, secretKey, sessionToken)
	}

	state.wsGameLift = wsGameLift
	err := state.wsGameLift.Connect(
		websocketUrl,
		state.processID,
		state.hostID,
		state.fleetID,
		authToken,
		sigV4QueryParameters,
	)
	if err != nil {
		return common.NewGameLiftError(common.LocalConnectionFailed, "", err.Error())
	}
	return nil
}

func getSigV4QueryParameters(awsRegion, accessKey, secretKey, sessionToken string) map[string]string {
	awsCredentials := security.AwsCredentials{AccessKey: accessKey, SecretKey: secretKey, SessionToken: sessionToken}
	queryParamsToSign := map[string]string{
		common.ComputeIDKey: state.hostID,
		common.FleetIDKey:   state.fleetID,
		common.PidKey:       state.processID,
	}

	sigV4Parameters := security.SigV4Parameters{
		AwsRegion:      awsRegion,
		AwsCredentials: awsCredentials,
		QueryParams:    queryParamsToSign,
		RequestTime:    time.Now().UTC(),
	}

	sigV4QueryParameters, err := security.GenerateSigV4QueryParameters(sigV4Parameters)
	if err != nil {
		log.Fatalf("Error generating SigV4 query string: %v\n", err)
	}
	return sigV4QueryParameters
}

func (state *gameLiftServerState) processReady(params *ProcessParameters) error {
	if params == nil {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	var res message.ResponseMessage
	state.parameters = params
	req := request.NewActivateServerProcess(
		common.SdkVersion,
		common.SdkLanguage,
		params.Port,
	)
	req.LogPaths = params.LogParameters.LogPaths

	// Wait for response from ActivateServerProcess() request
	err := state.wsGameLift.HandleRequest(req, &res, ActivateServerProcessRequestTimeoutInSeconds)

	if err != nil {
		return common.NewGameLiftError(common.ProcessNotReady, "", err.Error())
	}
	state.isReadyProcess.Store(true)
	state.shutdown = make(chan bool)
	go state.startHealthCheck(state.shutdown)
	return nil
}

func (state *gameLiftServerState) processEnding() error {
	err := state.wsGameLift.SendMessage(request.NewTerminateServerProcess())
	if err != nil {
		return common.NewGameLiftError(common.ProcessEndingFailed, "", err.Error())
	}
	state.stopServerProcess()

	return nil
}

func (state *gameLiftServerState) activateGameSession() error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	req := request.NewActivateGameSession(state.gameSessionID)
	err := state.wsGameLift.SendMessage(req)
	return err
}

func (state *gameLiftServerState) updatePlayerSessionCreationPolicy(policy *model.PlayerSessionCreationPolicy) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	if policy == nil {
		return common.NewGameLiftError(common.BadRequestException, "", "")
	}
	req := request.NewUpdatePlayerSessionCreationPolicy(state.gameSessionID, *policy)
	err := state.wsGameLift.SendMessage(req)
	return err
}

func (state *gameLiftServerState) getGameSessionID() (string, error) {
	return state.gameSessionID, nil
}

// getTerminationTime - returns number of seconds that have elapsed since Unix epoch time begins (00:00:00 UTC Jan 1 1970).
func (state *gameLiftServerState) getTerminationTime() (int64, error) {
	if state.terminationTime == 0 {
		return 0, common.NewGameLiftError(common.TerminationTimeNotSet, "", "")
	}
	return state.terminationTime, nil
}

func (state *gameLiftServerState) acceptPlayerSession(playerSessionID string) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	req := request.NewAcceptPlayerSession(state.gameSessionID, playerSessionID)
	err := state.wsGameLift.SendMessage(req)
	return err
}

func (state *gameLiftServerState) removePlayerSession(playerSessionID string) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	req := request.NewRemovePlayerSession(state.gameSessionID, playerSessionID)
	err := state.wsGameLift.SendMessage(req)
	return err
}

func (state *gameLiftServerState) describePlayerSessions(req *request.DescribePlayerSessionsRequest) (result.DescribePlayerSessionsResult, error) {
	var playerSessionResult result.DescribePlayerSessionsResult
	if !state.isReadyProcess.Load() {
		return playerSessionResult, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return playerSessionResult, common.NewGameLiftError(common.BadRequestException, "", "")
	}
	err := state.wsGameLift.HandleRequest(req, &playerSessionResult, state.serviceCallTimeout)
	return playerSessionResult, err
}

func (state *gameLiftServerState) startMatchBackfill(req *request.StartMatchBackfillRequest) (result.StartMatchBackfillResult, error) {
	var startMatchBackfillResult result.StartMatchBackfillResult
	if !state.isReadyProcess.Load() {
		return startMatchBackfillResult, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return startMatchBackfillResult, common.NewGameLiftError(common.BadRequestException, "", "")
	}
	err := state.wsGameLift.HandleRequest(req, &startMatchBackfillResult, state.serviceCallTimeout)
	return startMatchBackfillResult, err
}

func (state *gameLiftServerState) stopMatchBackfill(req *request.StopMatchBackfillRequest) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return common.NewGameLiftError(common.BadRequestException, "", "")
	}
	err := state.wsGameLift.SendMessage(req)
	return err
}

func (state *gameLiftServerState) getComputeCertificate() (result.GetComputeCertificateResult, error) {
	lg.Debugf("Calling GetComputeCertificate")
	var res result.GetComputeCertificateResult
	if !state.isReadyProcess.Load() {
		return res, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	err := state.wsGameLift.HandleRequest(request.NewGetComputeCertificate(), &res, state.serviceCallTimeout)
	return res, err
}

func (state *gameLiftServerState) getRoleCredentialsFromCache(roleArn string) (result.GetFleetRoleCredentialsResult, bool) {
	state.mtx.Lock()
	defer state.mtx.Unlock()
	if previousResult, ok := state.fleetRoleResultCache[roleArn]; ok {
		timeToLive := time.Duration(previousResult.Expiration-time.Now().UnixMilli()) * time.Millisecond
		if timeToLive > common.InstanceRoleCredentialTTL {
			return previousResult, true
		}
		delete(state.fleetRoleResultCache, roleArn)
	}
	return result.GetFleetRoleCredentialsResult{}, false
}

func (state *gameLiftServerState) getFleetRoleCredentials(
	req *request.GetFleetRoleCredentialsRequest,
) (result.GetFleetRoleCredentialsResult, error) {
	lg.Debugf("Calling GetFleetRoleCredentials")
	if !state.onManagedEC2 || req == nil {
		return result.GetFleetRoleCredentialsResult{},
			common.NewGameLiftError(common.BadRequestException, "", "")
	}
	res, ok := state.getRoleCredentialsFromCache(req.RoleArn)
	if ok {
		return res, nil
	}
	// If role session name was not provided, default to fleetId-hostId
	if req.RoleSessionName == "" {
		req.RoleSessionName = fmt.Sprintf("%s-%s", state.fleetID, state.hostID)
		if len(req.RoleSessionName) > common.RoleSessionNameMaxLength {
			req.RoleSessionName = req.RoleSessionName[:common.RoleSessionNameMaxLength]
		}
	}
	// Role session name cannot be over 64 chars (enforced by IAM's AssumeRole API)
	if len(req.RoleSessionName) > common.RoleSessionNameMaxLength {
		return res, common.NewGameLiftError(common.BadRequestException, "", "")
	}

	if !state.isReadyProcess.Load() {
		return res, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}

	err := state.wsGameLift.HandleRequest(req, &res, state.serviceCallTimeout)
	if err != nil {
		return res, err
	}
	if res.AccessKeyID == "" {
		state.onManagedEC2 = false
		return res, common.NewGameLiftError(common.BadRequestException, "", "")
	}

	state.mtx.Lock()
	defer state.mtx.Unlock()
	state.fleetRoleResultCache[req.RoleArn] = res

	return res, err
}

func (state *gameLiftServerState) destroy() error {
	state.stopServerProcess()
	return state.wsGameLift.Disconnect()
}

func (state *gameLiftServerState) stopServerProcess() {
	if state.isReadyProcess.CompareAndSwap(true, false) {
		if isChannelOpen(state.shutdown) && state.shutdown != nil {
			close(state.shutdown)
		}
	}
}

func (state *gameLiftServerState) startHealthCheck(done <-chan bool) {
	lg.Debugf("HealthCheck thread started.")
	for state.isReadyProcess.Load() {
		timeout := time.After(state.getNextHealthCheckIntervalSeconds())
		go state.heartbeatServerProcess(done)
		select {
		case <-timeout:
			continue
		case <-done:
			return
		}
	}
}

func (state *gameLiftServerState) heartbeatServerProcess(done <-chan bool) {
	res := make(chan bool)
	go func(res chan<- bool) {
		if state.parameters != nil && state.parameters.OnHealthCheck != nil {
			lg.Debugf("Reporting health using the OnHealthCheck callback.")
			res <- state.parameters.OnHealthCheck()
		} else {
			close(res)
		}
	}(res)
	timeout := time.After(state.healthCheckTimeout)
	status := false
	select {
	case <-timeout:
		lg.Debugf("Timed out waiting for health response from the server process. Reporting as unhealthy.")
		status = false
	case status = <-res:
		lg.Debugf("Received health response from the server process: %v", status)
	case <-done:
		return
	}
	var response message.Message
	err := state.wsGameLift.HandleRequest(
		request.NewHeartbeatServerProcess(status),
		&response,
		state.serviceCallTimeout,
	)
	if err != nil {
		lg.Warnf("Could not send health status: %s", err)
	}
}

// getNextHealthCheckIntervalSeconds - return a healthCheck interval +/- a random value
// between [- defaultJitterIntervalMs, defaultJitterIntervalMs].
//
//nolint:gosec // weak math random generator is enough in this case
func (state *gameLiftServerState) getNextHealthCheckIntervalSeconds() time.Duration {
	jitterMs := 2*localRnd.Int63n(state.defaultJitterIntervalMs) - state.defaultJitterIntervalMs
	return state.healthCheckInterval - time.Duration(jitterMs)*time.Millisecond
}

// OnStartGameSession handler for message.CreateGameSessionMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnStartGameSession(session *model.GameSession) {
	if session == nil {
		lg.Warnf("OnStartGameSession was called with nil game session")
		return
	}
	// Inject data that already exists on the server
	session.FleetID = state.fleetID
	lg.Debugf("server got the startGameSession signal. GameSession : %s", session.GameSessionID)
	if !state.isReadyProcess.Load() {
		lg.Debugf("Got a game session on inactive process. Ignoring.")
		return
	}
	state.gameSessionID = session.GameSessionID
	if state.parameters != nil && state.parameters.OnStartGameSession != nil {
		state.parameters.OnStartGameSession(*session)
	}
}

// OnUpdateGameSession - handler for message.UpdateGameSessionMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnUpdateGameSession(
	gameSession *model.GameSession,
	updateReason *model.UpdateReason,
	backfillTicketID string,
) {
	if gameSession == nil {
		lg.Warnf("OnUpdateGameSession was called with nil game session")
		return
	}
	lg.Debugf("ServerState got the updateGameSession signal. GameSession : %s", gameSession.GameSessionID)
	if !state.isReadyProcess.Load() {
		lg.Warnf("Got an updated game session on inactive process.")
		return
	}
	if updateReason == nil {
		lg.Warnf("OnUpdateGameSession was called with nil update reason")
		return
	}
	if state.parameters != nil && state.parameters.OnUpdateGameSession != nil {
		state.parameters.OnUpdateGameSession(
			model.UpdateGameSession{
				GameSession:      *gameSession,
				UpdateReason:     updateReason,
				BackfillTicketID: backfillTicketID,
			},
		)
	}
}

// OnTerminateProcess - handler for message.TerminateProcessMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnTerminateProcess(terminationTime int64) {
	// terminationTime is milliseconds that have elapsed since Unix epoch time begins (00:00:00 UTC Jan 1 1970).
	state.terminationTime = terminationTime / 1000
	lg.Debugf("ServerState got the terminateProcess signal. termination time : %d", state.terminationTime)
	if state.parameters != nil && state.parameters.OnProcessTerminate != nil {
		state.parameters.OnProcessTerminate()
	}
}

// OnRefreshConnection - callback function that the Gamelift service invokes when
// the server process need to refresh current websocket connection.
func (state *gameLiftServerState) OnRefreshConnection(refreshConnectionEndpoint, authToken string) {
	err := state.wsGameLift.Connect(
		refreshConnectionEndpoint,
		state.processID,
		state.hostID,
		state.fleetID,
		authToken,
		nil,
	)
	if err != nil {
		lg.Errorf("Failed to refresh websocket connection. The GameLift SDK will try again each minute "+
			"until the refresh succeeds, or the websocket is forcibly closed: %s", err)
	}
}

func isChannelOpen(ch <-chan bool) bool {
	select {
	case <-ch:
		return false
	default:
	}
	return true
}
