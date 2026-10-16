package driver

import (
	"encoding/json"

	"github.com/LMgateways/iotserver-sdk-go/model"
)

// handleUplink 处理设备上行报文（由 internal/client 的 MQTT 订阅回调触发）。
//
// 报文一律是 JSON，按主题末段区分类型：
//
//	properties  {"temp": 23.5, "rpm": 1450}          → 属性上报
//	events      {"alarm": {"code": "E01"}}            → 事件上报
//	status      {"online": true}                      → 在线状态 / 心跳
//
// 设备用序列号（DeviceSn）标识，这里反查 deviceId 后调 SDK 上报。
func (dr *MQTTProtocolDriver) handleUplink(deviceSn, suffix string, payload []byte) {
	device, ok := dr.deviceBySn(deviceSn)
	if !ok {
		dr.sd.GetLogger().Warnf("uplink from unknown device sn=%s topic=%s (device not registered in platform)", deviceSn, suffix)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		dr.sd.GetLogger().Errorf("parse uplink payload failed device=%s suffix=%s: %v", deviceSn, suffix, err)
		return
	}

	switch suffix {
	case "properties":
		dr.reportProperties(device, body)
	case "events":
		dr.reportEvents(device, body)
	case "status":
		dr.updateStatus(device, body)
	default:
		dr.sd.GetLogger().Warnf("unknown uplink suffix=%s device=%s", suffix, deviceSn)
	}
}

// reportProperties 上报属性：{"temp": 23.5} → PropertyReport{temp: {value:23.5, time:now}}
func (dr *MQTTProtocolDriver) reportProperties(device model.Device, body map[string]interface{}) {
	data := make(map[string]model.PropertyData, len(body))
	for code, value := range body {
		if code == "ts" || code == "deviceSn" {
			continue
		}
		data[code] = model.NewPropertyData(value)
	}
	if len(data) == 0 {
		return
	}
	if _, err := dr.sd.PropertyReport(device.Id, model.NewPropertyReport(false, data)); err != nil {
		dr.sd.GetLogger().Errorf("property report failed device=%s: %v", device.Id, err)
		return
	}
	// 收到属性即视为在线
	_ = dr.sd.Online(device.Id)
	dr.sd.GetLogger().Infof("property reported device=%s count=%d", device.Id, len(data))
}

// reportEvents 上报事件：{"alarm": {"code": "E01", "msg": "超温"}}
//   - 值为对象时，取其中的 outputParams（没有则整对象作为输出参数）
//   - 值为标量时，整值作为输出参数
func (dr *MQTTProtocolDriver) reportEvents(device model.Device, body map[string]interface{}) {
	for code, raw := range body {
		params := toOutputParams(raw)
		if _, err := dr.sd.EventReport(device.Id, model.NewEventReport(false, model.NewEventData(code, params))); err != nil {
			dr.sd.GetLogger().Errorf("event report failed device=%s code=%s: %v", device.Id, code, err)
			continue
		}
		dr.sd.GetLogger().Infof("event reported device=%s code=%s", device.Id, code)
	}
}

// updateStatus 心跳 / 在线状态：{"online": true, "fw": "1.0"}
func (dr *MQTTProtocolDriver) updateStatus(device model.Device, body map[string]interface{}) {
	online, ok := body["online"].(bool)
	if !ok {
		// 没带 online 字段就当成心跳，视为在线
		online = true
	}
	if online {
		_ = dr.sd.Online(device.Id)
		dr.sd.GetLogger().Infof("device online: %s", device.Id)
	} else {
		_ = dr.sd.Offline(device.Id)
		dr.sd.GetLogger().Infof("device offline: %s", device.Id)
	}
}

// deviceBySn 按设备序列号查设备（平台的 deviceId 与设备侧可见的 SN 不是一回事，
// 所以上下行统一用 SN 做标识，这里反查）。
func (dr *MQTTProtocolDriver) deviceBySn(deviceSn string) (model.Device, bool) {
	for _, d := range dr.sd.GetDeviceList() {
		if d.DeviceSn == deviceSn {
			return d, true
		}
	}
	return model.Device{}, false
}

// toOutputParams 事件输出参数归一化。
func toOutputParams(raw interface{}) map[string]interface{} {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"value": raw}
	}
	// 事件报文里常见 {"code": "...", "msg": "..."}，也可显式给 outputParams
	if p, ok := m["outputParams"].(map[string]interface{}); ok {
		return p
	}
	return m
}