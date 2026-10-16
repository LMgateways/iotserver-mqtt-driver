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

// Package client 驱动内部的 MQTT 客户端：连接内嵌 broker，订阅设备上行主题，
// 把收到的报文交给驱动解析并上报平台；同时用于向设备发布下行消息。
package client

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/LMgateways/iotserver-sdk-go/service"

	"github.com/LMgateways/iotserver-mqtt-driver/constants"
	"github.com/LMgateways/iotserver-mqtt-driver/internal/topic"

	drivercfg "github.com/LMgateways/iotserver-mqtt-driver/config"
)

var driverService *service.DriverService

// UplinkHandler 上行报文处理：驱动实现它，把设备报文转成平台物模型消息。
// deviceSn 为设备序列号，suffix 为主题末段（properties / events 等），payload 为原始报文。
type UplinkHandler func(deviceSn string, suffix string, payload []byte)

// NewMQTTClient 连上内嵌 broker，订阅设备上行主题。
//
// 连不上时只记录日志、不 panic：驱动退化成"只能下发、不能接收"，
// 比整个进程崩掉更利于排查（broker 可能需要更长时间启动）。
func NewMQTTClient(sd *service.DriverService, handler UplinkHandler) mqtt.Client {
	driverService = sd

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", "127.0.0.1", drivercfg.GetConfig().BrokerPort))
	opts.SetClientID(constants.MQTTInnerClientId)
	opts.SetUsername(constants.MQTTInnerUsername)
	opts.SetPassword(constants.MQTTInnerPassword(sd))
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.OnConnect = func(c mqtt.Client) {
		sd.GetLogger().Info("internal mqtt client connected")
		subscribeUplink(c, sd, handler)
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		sd.GetLogger().Infof("internal mqtt client connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		sd.GetLogger().Errorf("connect internal mqtt broker failed: %v", token.Error())
	}

	return client
}

// subscribeUplink 订阅所有设备的上行主题（用 + 通配，设备上下线都不用改订阅）。
func subscribeUplink(c mqtt.Client, sd *service.DriverService, handler UplinkHandler) {
	filters := []string{
		topic.UplinkRoot() + "/+/properties",
		topic.UplinkRoot() + "/+/events",
		topic.UplinkRoot() + "/+/status",
	}
	for _, f := range filters {
		filter := f // capture
		token := c.Subscribe(filter, 1, func(_ mqtt.Client, msg mqtt.Message) {
			deviceSn, suffix, ok := topic.ParseUplink(msg.Topic())
			if !ok {
				sd.GetLogger().Warnf("cannot parse uplink topic: %s", msg.Topic())
				return
			}
			handler(deviceSn, suffix, msg.Payload())
		})
		token.WaitTimeout(5 * time.Second)
		if err := token.Error(); err != nil {
			sd.GetLogger().Errorf("subscribe %s failed: %v", filter, err)
		} else {
			sd.GetLogger().Infof("subscribed uplink topic: %s", filter)
		}
	}
}