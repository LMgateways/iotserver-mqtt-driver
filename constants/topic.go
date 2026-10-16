package constants

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/LMgateways/iotserver-sdk-go/service"
)

// 驱动内部 MQTT 客户端凭据。
//
// 该客户端只在内嵌 broker 上收发消息（订阅设备上行、发布下行），不对外暴露。
// 密码不再硬编码：由驱动实例的 CustomParam 派生（每个实例不同），
// broker 鉴权钩子与客户端调同一个函数，双方自然一致。
const (
	MQTTInnerClientId = "iotserver-mqtt-driver-internal"
	MQTTInnerUsername = "iotserver-mqtt-driver-internal"

	// BrokerPort 内嵌 broker 的监听端口（设备接入端口）
	BrokerPort = 1883
)

// MQTTInnerPassword 派生内部客户端密码：sha256(实例 CustomParam)。
// CustomParam 是平台按实例下发的自定义参数，天然每实例唯一；
// 未配置时退回固定串（本地调试场景）。
func MQTTInnerPassword(sd *service.DriverService) string {
	salt := ""
	if sd != nil {
		salt = sd.GetCustomParam()
	}
	if salt == "" {
		salt = "iotserver-mqtt-driver-local"
	}
	sum := sha256.Sum256([]byte("iotserver-mqtt-driver:" + salt))
	return hex.EncodeToString(sum[:])
}
