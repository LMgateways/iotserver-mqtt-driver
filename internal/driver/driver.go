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

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/LMgateways/iotserver-sdk-go/commons"
	"github.com/LMgateways/iotserver-sdk-go/model"
	"github.com/LMgateways/iotserver-sdk-go/service"

	"github.com/LMgateways/iotserver-mqtt-driver/config"
	"github.com/LMgateways/iotserver-mqtt-driver/internal/client"
	"github.com/LMgateways/iotserver-mqtt-driver/internal/server"
	"github.com/LMgateways/iotserver-mqtt-driver/internal/topic"
)

// MQTTProtocolDriver MQTT 协议驱动。
//
// 职责：
//   - 内嵌一个 MQTT broker（默认 0.0.0.0:1883），设备直连该 broker 上报数据；
//   - broker 收到的报文由 internal/server 的钩子解析，转成平台物模型消息上报；
//   - 平台下发的属性设置 / 属性查询 / 服务调用，通过内部 MQTT 客户端转发给设备。
type MQTTProtocolDriver struct {
	sd         *service.DriverService
	mqttClient mqtt.Client

	mu      sync.RWMutex
	devices map[string]model.Device // deviceId → 设备（平台通知维护）
}

// CloudPluginNotify 云插件启动/停止通知。
// 本驱动不依赖云插件，收到通知只记录日志。
func (dr *MQTTProtocolDriver) CloudPluginNotify(ctx context.Context, t commons.CloudPluginNotifyType, name string) error {
	dr.sd.GetLogger().Infof("cloud plugin notify: type=%s name=%s", t, name)
	return nil
}

// DeviceNotify 设备添加/修改/删除通知。
// 维护驱动侧设备表；设备删除时把它置为离线，避免平台上残留"在线"状态。
func (dr *MQTTProtocolDriver) DeviceNotify(ctx context.Context, t commons.DeviceNotifyType, deviceId string, device model.Device) error {
	switch t {
	case commons.DeviceAddNotify:
		dr.putDevice(device)
		dr.sd.GetLogger().Infof("device add: id=%s sn=%s", deviceId, device.DeviceSn)
	case commons.DeviceUpdateNotify:
		dr.putDevice(device)
		dr.sd.GetLogger().Infof("device update: id=%s sn=%s", deviceId, device.DeviceSn)
	case commons.DeviceDeleteNotify:
		dr.removeDevice(deviceId)
		dr.sd.Offline(deviceId)
		dr.sd.GetLogger().Infof("device delete: id=%s", deviceId)
	default:
		dr.sd.GetLogger().Warnf("unknown device notify type: %s", t)
	}
	return nil
}

// ProductNotify 产品添加/修改/删除通知。
// 物模型由 SDK 缓存，驱动侧无需额外处理，记录日志即可。
func (dr *MQTTProtocolDriver) ProductNotify(ctx context.Context, t commons.ProductNotifyType, productId string, product model.Product) error {
	dr.sd.GetLogger().Infof("product notify: type=%s id=%s", t, productId)
	return nil
}

// Stop 驱动退出通知：把所有设备置为离线，断开内部 MQTT 客户端。
func (dr *MQTTProtocolDriver) Stop(ctx context.Context) error {
	for _, device := range dr.sd.GetDeviceList() {
		dr.sd.Offline(device.Id)
	}
	if dr.mqttClient != nil && dr.mqttClient.IsConnected() {
		dr.mqttClient.Disconnect(250)
	}
	dr.sd.GetLogger().Info("mqtt driver stopped")
	return nil
}

// HandlePropertySet 设备属性设置：转发到设备的 MQTT 下行主题。
func (dr *MQTTProtocolDriver) HandlePropertySet(ctx context.Context, deviceId string, data model.PropertySet) error {
	device, ok := dr.sd.GetDeviceById(deviceId)
	if !ok {
		return fmt.Errorf("device [%s] not found", deviceId)
	}
	return dr.publishDownlink(device, "set", data)
}

// HandlePropertyGet 设备属性查询：转发到设备，由设备自行上报查询结果。
func (dr *MQTTProtocolDriver) HandlePropertyGet(ctx context.Context, deviceId string, data model.PropertyGet) error {
	device, ok := dr.sd.GetDeviceById(deviceId)
	if !ok {
		return fmt.Errorf("device [%s] not found", deviceId)
	}
	return dr.publishDownlink(device, "get", data)
}

// HandleServiceExecute 设备服务调用：转发到设备。
func (dr *MQTTProtocolDriver) HandleServiceExecute(ctx context.Context, deviceId string, data model.ServiceExecuteRequest) error {
	device, ok := dr.sd.GetDeviceById(deviceId)
	if !ok {
		return fmt.Errorf("device [%s] not found", deviceId)
	}
	return dr.publishDownlink(device, "service", data)
}

// publishDownlink 把下行消息发布到 {DownlinkRoot}/{deviceSn}/{action}。
// 设备订阅自己的下行主题即可收到平台的设置 / 查询 / 服务调用。
func (dr *MQTTProtocolDriver) publishDownlink(device model.Device, action string, data interface{}) error {
	if dr.mqttClient == nil || !dr.mqttClient.IsConnected() {
		return fmt.Errorf("internal mqtt client not connected, cannot send %s to device %s", action, device.Id)
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal downlink payload: %w", err)
	}
	topic := topic.DownlinkTopic(device.DeviceSn, action)
	token := dr.mqttClient.Publish(topic, 1, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish %s to topic %s: timeout", action, topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish %s to topic %s: %w", action, topic, err)
	}
	dr.sd.GetLogger().Infof("downlink %s sent to device %s (topic=%s)", action, device.Id, topic)
	return nil
}

func (dr *MQTTProtocolDriver) putDevice(device model.Device) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	if dr.devices == nil {
		dr.devices = make(map[string]model.Device)
	}
	dr.devices[device.Id] = device
}

func (dr *MQTTProtocolDriver) removeDevice(deviceId string) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	delete(dr.devices, deviceId)
}

// NewMQTTProtocolDriver 构造 MQTT 协议驱动：先起内嵌 broker，再建内部 MQTT 客户端。
func NewMQTTProtocolDriver(sd *service.DriverService) *MQTTProtocolDriver {
	dr := &MQTTProtocolDriver{
		sd:      sd,
		devices: make(map[string]model.Device),
	}

	// 主题根（默认 iot，可由自定义参数 MQTT.TopicRoot 覆盖）
	topic.SetRoot(config.GetConfig().TopicRoot)

	// 内嵌 MQTT broker（设备接入端口，默认 1883）
	go server.NewMQTTService(sd).Start()

	// 等 broker 起来（否则内部客户端连不上）
	time.Sleep(2 * time.Second)

	// 内部 MQTT 客户端：订阅设备上行报文，交 dr.handleUplink 解析上报
	dr.mqttClient = client.NewMQTTClient(sd, dr.handleUplink)
	return dr
}