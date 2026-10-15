/*******************************************************************************
 * Copyright 2017.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/winc-link/hummingbird-sdk-go/service"

	"github.com/DrmagicE/gmqtt/config"
	"github.com/DrmagicE/gmqtt/pkg/pidfile"
	"github.com/DrmagicE/gmqtt/server"

	_ "github.com/DrmagicE/gmqtt/persistence"
	_ "github.com/DrmagicE/gmqtt/plugin/prometheus"
	_ "github.com/DrmagicE/gmqtt/topicalias/fifo"
)

var GlobalDriverService *service.DriverService

type MQTTServer struct {
}

func NewMQTTService(sd *service.DriverService) *MQTTServer {
	GlobalDriverService = sd
	return &MQTTServer{}
}

func (m *MQTTServer) Start() {
	c := config.DefaultConfig()
	c.Listeners = listeners()

	if c.PidFile != "" {
		pid, err := pidfile.New(c.PidFile)
		if err != nil {
			GlobalDriverService.GetLogger().Errorf("create pid file failed: %v", err)
			return
		}
		defer pid.Remove()
	}

	tcpListeners, websockets, err := GetListeners(c)
	if err != nil {
		GlobalDriverService.GetLogger().Errorf("init mqtt listeners failed: %v", err)
		return
	}

	l, err := c.GetLogger(c.Log)
	if err != nil {
		GlobalDriverService.GetLogger().Errorf("init mqtt logger failed: %v", err)
		return
	}

	//设置Hooks
	hooks := server.Hooks{
		OnAccept:            OnAccept,
		OnStop:              OnStop,
		OnSubscribe:         OnSubscribe,
		OnSubscribed:        OnSubscribed,
		OnUnsubscribe:       OnUnsubscribe,
		OnUnsubscribed:      OnUnsubscribed,
		OnMsgArrived:        OnMsgArrived,
		OnBasicAuth:         OnBasicAuth,
		OnEnhancedAuth:      OnEnhancedAuth,
		OnReAuth:            OnReAuth,
		OnConnected:         OnConnected,
		OnSessionCreated:    OnSessionCreated,
		OnSessionResumed:    OnSessionResumed,
		OnSessionTerminated: OnSessionTerminated,
		OnDelivered:         OnDelivered,
		OnClosed:            OnClosed,
		OnMsgDropped:        OnMsgDropped,
		OnWillPublish:       OnWillPublish,
		OnWillPublished:     OnWillPublished,
	}
	s := server.New(
		server.WithConfig(c),
		server.WithTCPListener(tcpListeners...),
		server.WithWebsocketServer(websockets...),
		server.WithHook(hooks),
		server.WithLogger(l),
	)

	err = s.Init()
	if err != nil {
		GlobalDriverService.GetLogger().Errorf("init mqtt broker failed: %v", err)
		return
	}

	// 启动 broker（Run 阻塞，放到调用方的 goroutine 里）
	GlobalDriverService.GetLogger().Infof("mqtt broker listening on %s", listenerAddrs(c.Listeners))
	if err = s.Run(); err != nil {
		GlobalDriverService.GetLogger().Errorf("mqtt broker exited: %v", err)
	}
}

// listenerAddrs 拼接监听地址，便于启动日志排查端口占用。
func listenerAddrs(ls []*config.ListenerConfig) string {
	addrs := make([]string, 0, len(ls))
	for _, l := range ls {
		addrs = append(addrs, l.Address)
	}
	return strings.Join(addrs, ",")
}

func GetListeners(c config.Config) (tcpListeners []net.Listener, websockets []*server.WsServer, err error) {
	for _, v := range c.Listeners {
		var ln net.Listener
		if v.Websocket != nil {
			ws := &server.WsServer{
				Server: &http.Server{Addr: v.Address},
				Path:   v.Websocket.Path,
			}
			if v.TLSOptions != nil {
				ws.KeyFile = v.Key
				ws.CertFile = v.Cert
			}
			websockets = append(websockets, ws)
			continue
		}
		if v.TLSOptions != nil {
			var cert tls.Certificate
			cert, err = tls.LoadX509KeyPair(v.Cert, v.Key)
			if err != nil {
				return
			}
			ln, err = tls.Listen("tcp", v.Address, &tls.Config{
				Certificates: []tls.Certificate{cert},
			})
		} else {
			ln, err = net.Listen("tcp", v.Address)
		}
		tcpListeners = append(tcpListeners, ln)
	}
	return
}
