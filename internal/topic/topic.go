// Package topic 驱动与设备之间的 MQTT 主题约定。
//
// 上下行主题都只用设备序列号（DeviceSn）做标识，方便设备侧实现：
//
//	上行（设备 → 驱动）  {root}/up/{deviceSn}/properties  {"temp": 23.5, "rpm": 1450}
//	                     {root}/up/{deviceSn}/events      {"alarm": {"code": "E01"}}
//	                     {root}/up/{deviceSn}/status      {"online": true}
//	下行（驱动 → 设备）  {root}/down/{deviceSn}/set       属性设置
//	                     {root}/down/{deviceSn}/get       属性查询
//	                     {root}/down/{deviceSn}/service   服务调用
//
// root 默认 "iot"，可用自定义参数 MQTT.TopicRoot 覆盖。
package topic

import (
	"strings"
	"sync"
)

var (
	mu   sync.RWMutex
	root = "iot"
)

// SetRoot 设置主题根（驱动启动时从自定义参数读取）。
func SetRoot(r string) {
	r = strings.Trim(strings.TrimSpace(r), "/")
	if r == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	root = r
}

// Root 当前主题根。
func Root() string {
	mu.RLock()
	defer mu.RUnlock()
	return root
}

// UplinkRoot 上行主题前缀：{root}/up
func UplinkRoot() string { return Root() + "/up" }

// DownlinkRoot 下行主题前缀：{root}/down
func DownlinkRoot() string { return Root() + "/down" }

// ParseUplink 解析上行主题 {root}/up/{deviceSn}/{suffix}。
func ParseUplink(t string) (deviceSn, suffix string, ok bool) {
	prefix := UplinkRoot() + "/"
	if !strings.HasPrefix(t, prefix) {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(t, prefix), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// DownlinkTopic 拼下行主题 {root}/down/{deviceSn}/{action}。
func DownlinkTopic(deviceSn, action string) string {
	return DownlinkRoot() + "/" + deviceSn + "/" + action
}
