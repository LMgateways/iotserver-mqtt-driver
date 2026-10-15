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
	"fmt"

	"github.com/DrmagicE/gmqtt/config"

	drivercfg "github.com/iotserver-org/iotserver-mqtt-driver/config"
)

// listeners 内嵌 broker 的监听地址。
//
// 默认只开一个 TCP 端口（设备接入口）。端口由自定义参数 BrokerPort 决定，
// 平台创建驱动实例时可通过自定义参数覆盖（注意同时要把该端口映射出容器）。
func listeners() []*config.ListenerConfig {
	port := drivercfg.GetConfig().BrokerPort
	return []*config.ListenerConfig{
		{
			Address:    fmt.Sprintf("0.0.0.0:%d", port),
			TLSOptions: nil,
			Websocket:  nil,
		},
		// TLS（8883）与 WebSocket 接入按需打开：
		//{
		//	Address: "0.0.0.0:8883",
		//	TLSOptions: &config.TLSOptions{Cert: "/etc/driver/cert.pem", Key: "/etc/driver/key.pem"},
		//},
		//{
		//	Address:   "0.0.0.0:8083",
		//	Websocket: &config.WebsocketOptions{Path: "/mqtt"},
		//},
	}
}