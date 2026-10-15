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

// Package config 驱动自定义配置：平台在「驱动实例」编辑页的「自定义参数」里填 JSON，
// 驱动启动时由 SDK 透传（sd.GetCustomParam()），这里解析成结构体。
package config

import (
	"encoding/json"

	"github.com/winc-link/hummingbird-sdk-go/service"
)

var baseConfig *BaseConfig

// BaseConfig 驱动自定义参数。对应驱动实例编辑页里填的 JSON，例如：
//
//	{
//	  "TopicRoot": "iot",
//	  "BrokerPort": 1883,
//	  "AllowAnonymousDevice": true
//	}
type BaseConfig struct {
	// TopicRoot 主题根，默认 "iot"。
	// 上下行主题为 {TopicRoot}/up/{deviceSn}/properties 与 {TopicRoot}/down/{deviceSn}/set
	TopicRoot string
	// BrokerPort 内嵌 MQTT broker 的监听端口，默认 1883
	BrokerPort int
	// AllowAnonymousDevice 是否允许未在平台注册的设备直接连接 broker（调试用，默认 false）。
	// 关闭时只有平台已注册的设备（序列号匹配）能连上。
	AllowAnonymousDevice bool
}

func InitConfig(sd *service.DriverService) {
	customParam := sd.GetCustomParam()
	baseConfig = &BaseConfig{
		TopicRoot:  "iot",
		BrokerPort: 1883,
	}
	if customParam == "" {
		return
	}
	if err := json.Unmarshal([]byte(customParam), baseConfig); err != nil {
		// 自定义参数是用户手填的，格式错误时记录日志并退回默认值，不阻断驱动启动
		sd.GetLogger().Errorf("parse custom param failed, use defaults: %v", err)
		baseConfig = &BaseConfig{TopicRoot: "iot", BrokerPort: 1883}
		return
	}
	if baseConfig.TopicRoot == "" {
		baseConfig.TopicRoot = "iot"
	}
	if baseConfig.BrokerPort == 0 {
		baseConfig.BrokerPort = 1883
	}
}

// GetConfig 返回驱动自定义配置（InitConfig 之后一定非 nil）。
func GetConfig() *BaseConfig {
	if baseConfig == nil {
		return &BaseConfig{TopicRoot: "iot", BrokerPort: 1883}
	}
	return baseConfig
}