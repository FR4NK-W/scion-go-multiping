// Copyright 2024 ETH Zurich, OVGU Magdeburg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/daemon"
	"github.com/scionproto/scion/pkg/snet"
	"github.com/scionproto/scion/pkg/sock/reliable"
)

// hostContext contains the information needed to connect to the host's local SCION stack,
// i.e. the connection to sciond and dispatcher.
type hostContext struct {
	ia            addr.IA
	sciond        daemon.Connector
	dispatcher    reliable.Dispatcher
	hostInLocalAS net.IP
}

const (
	initTimeout = 1 * time.Second
)

var singletonHostContext hostContext
var initOnce sync.Once

// host initialises and returns the singleton hostContext.
func host() *hostContext {
	initOnce.Do(mustInitHostContext)
	return &singletonHostContext
}

func mustInitHostContext() {
	hostCtx, err := initHostContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing SCION host context: %v\n", err)
		os.Exit(1)
	}
	singletonHostContext = hostCtx
}

func initHostContext() (hostContext, error) {
	ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	dispatcher, err := findDispatcher()
	if err != nil {
		return hostContext{}, err
	}
	sciondConn, err := findSciond(ctx)
	if err != nil {
		return hostContext{}, err
	}
	localIA, err := sciondConn.LocalIA(ctx)
	if err != nil {
		return hostContext{}, err
	}
	hostInLocalAS, err := findAnyHostInLocalAS(ctx, sciondConn)
	if err != nil {
		return hostContext{}, err
	}
	return hostContext{
		ia:            addr.IA(localIA),
		sciond:        sciondConn,
		dispatcher:    dispatcher,
		hostInLocalAS: hostInLocalAS,
	}, nil
}

func findSciond(ctx context.Context) (daemon.Connector, error) {
	address, ok := os.LookupEnv("SCION_DAEMON_ADDRESS")
	if !ok {
		address = daemon.DefaultAPIAddress
	}
	sciondConn, err := daemon.NewService(address).Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to SCIOND at %s (override with SCION_DAEMON_ADDRESS): %w", address, err)
	}
	return sciondConn, nil
}

func findDispatcher() (reliable.Dispatcher, error) {
	path := getDispatcherPath()
	dispatcher := reliable.NewDispatcher(path)
	return dispatcher, nil
}

// findAnyHostInLocalAS returns the IP address of some (infrastructure) host in the local AS.
func findAnyHostInLocalAS(ctx context.Context, sciondConn daemon.Connector) (net.IP, error) {
	addr, err := daemon.TopoQuerier{Connector: sciondConn}.UnderlayAnycast(ctx, addr.SvcCS)
	if err != nil {
		return nil, err
	}
	return addr.IP, nil
}

func (h *hostContext) queryPaths(ctx context.Context, dst addr.IA) ([]snet.Path, error) {
	flags := daemon.PathReqFlags{Refresh: false, Hidden: false}
	snetPaths, err := h.sciond.Paths(ctx, addr.IA(dst), 0, flags)
	if err != nil {
		return nil, err
	}
	return snetPaths, nil
}

// Put here to be replaced in the scion-v12 branch with the dispatcherless connection
func newSCIONConn(ctx context.Context, handler scmpHandler, localIA addr.IA, localAddr net.UDPAddr) (snet.PacketConn, uint16, error) {
	dispSockerPath := getDispatcherPath()
	svc := snet.DefaultPacketDispatcherService{
		Dispatcher:  reliable.NewDispatcher(dispSockerPath),
		SCMPHandler: handler,
	}
	conn, port, err := svc.Register(ctx, localIA, &localAddr, addr.SvcNone)
	if err != nil {
		return nil, 0, err
	}

	return conn, port, nil
}
