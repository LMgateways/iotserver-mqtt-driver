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
	"context"
	"fmt"
	"net"

	"github.com/DrmagicE/gmqtt"
	"github.com/DrmagicE/gmqtt/pkg/packets"
	"github.com/DrmagicE/gmqtt/server"

	"github.com/LMgateways/iotserver-mqtt-driver/constants"
)

// OnAccept TCP连接建立时调用
// TCP连接限速，黑白名单等.
func OnAccept(ctx context.Context, conn net.Conn) bool {
	return true
}

// OnStop 当gmqtt退出时调用
func OnStop(ctx context.Context) {}

// OnSubscribe 收到订阅请求时调用
// 校验订阅是否合法
func OnSubscribe(ctx context.Context, client server.Client, req *server.SubscribeRequest) error {
	return nil
}

// OnSubscribed 订阅成功后调用
// 统计订阅报文数量
func OnSubscribed(ctx context.Context, client server.Client, subscription *gmqtt.Subscription) {}

// OnUnsubscribe 取消订阅时调用
// 校验是否允许取消订阅
func OnUnsubscribe(ctx context.Context, client server.Client, req *server.UnsubscribeRequest) error {
	return nil
}

// OnUnsubscribed 取消订阅成功后调用
// 统计订阅报文数
func OnUnsubscribed(ctx context.Context, client server.Client, topicName string) {}

// OnMsgArrived 收到消息发布报文时调用
// 校验发布权限，改写发布消息
func OnMsgArrived(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) error {
	return nil
}

// OnBasicAuth 收到连接请求报文时调用
//
// 鉴权规则：
//  1. 驱动内部客户端（订阅上行、发布下行）用派生密码放行——只在内网使用；
//  2. 设备连接用「设备序列号 + 设备密钥」校验：序列号必须是平台上已注册的设备，
//     密钥与平台上该设备的 Secret 一致，避免任意客户端冒充电表往平台灌数据。
func OnBasicAuth(ctx context.Context, client server.Client, req *server.ConnectRequest) (err error) {
	clientId := string(req.Connect.ClientID)
	username := string(req.Connect.Username)
	password := string(req.Connect.Password)

	// 1. 内部客户端
	if clientId == constants.MQTTInnerClientId && username == constants.MQTTInnerUsername &&
		GlobalDriverService != nil && password == constants.MQTTInnerPassword(GlobalDriverService) {
		return nil
	}

	// 2. 设备客户端：clientId / username 用设备序列号，password 用设备密钥
	if GlobalDriverService == nil {
		return fmt.Errorf("driver service not ready")
	}
	for _, dev := range GlobalDriverService.GetDeviceList() {
		if dev.DeviceSn != clientId && dev.DeviceSn != username {
			continue
		}
		// 平台未配密钥时放行（开发阶段）；配了就严格校验
		if dev.Secret == "" || password == dev.Secret {
			return nil
		}
		return fmt.Errorf("device %s auth failed: wrong secret", clientId)
	}
	return fmt.Errorf("device %s not registered on platform", clientId)
}

// OnEnhancedAuth 收到带有AuthMetho的连接请求报文时调用（V5特性）
// 客户端连接鉴权
func OnEnhancedAuth(ctx context.Context, client server.Client, req *server.ConnectRequest) (resp *server.EnhancedAuthResponse, err error) {
	return nil, nil
}

// OnReAuth 收到Auth报文时调用（V5特性）
// 客户端连接鉴权
func OnReAuth(ctx context.Context, client server.Client, auth *packets.Auth) (*server.AuthResponse, error) {
	return nil, nil
}

// OnConnected 客户端连接成功后调用
// 统计在线客户端数量
func OnConnected(ctx context.Context, client server.Client) {}

// OnSessionCreated
// 统计session数量
func OnSessionCreated(ctx context.Context, client server.Client) {}

// OnSessionResumed 客户端从旧session恢复后调用
// 统计session数量
func OnSessionResumed(ctx context.Context, client server.Client) {}

// OnSessionTerminated session删除后调用
// 统计session数量
func OnSessionTerminated(ctx context.Context, clientID string, reason server.SessionTerminatedReason) {
}

// OnDelivered 消息从broker投递到客户端后调用
func OnDelivered(ctx context.Context, client server.Client, msg *gmqtt.Message) {}

// OnClosed 统计在线客户端数量
func OnClosed(ctx context.Context, client server.Client, err error) {}

// OnMsgDropped 消息被丢弃时调用
func OnMsgDropped(ctx context.Context, clientID string, msg *gmqtt.Message, err error) {}

// OnWillPublish 发布遗嘱消息前
// 修改或丢弃遗嘱消息
func OnWillPublish(ctx context.Context, clientID string, req *server.WillMsgRequest) {}

// OnWillPublished 发布遗嘱消息后
func OnWillPublished(ctx context.Context, clientID string, msg *gmqtt.Message) {}
