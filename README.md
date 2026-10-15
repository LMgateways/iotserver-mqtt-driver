# iotserver-mqtt-driver

iotserver 物联网平台的 **MQTT 协议驱动**：驱动内嵌一个 MQTT broker，设备直连它上报数据，
驱动把报文转成平台物模型消息通过 gRPC 上报。

```
设备 ──MQTT──► 驱动（内嵌 broker）──gRPC──► 平台核心
                  ▲                           │
                  └────── 属性设置 / 服务调用 ──┘
```

## 一、平台侧怎么用

1. **构建镜像**（换成你的仓库地址）：

   ```bash
   make push IMAGE=registry.example.com/iotserver-mqtt-driver:v1.0.0
   ```

2. **平台「设备接入 → 驱动管理」**：
   - 「驱动市场」或「自定义」里添加驱动，镜像填上面的地址；
   - 安装后到「驱动实例」新建实例，**自定义参数**填：

     ```json
     {
       "TopicRoot": "iot",
       "BrokerPort": 1883
     }
     ```

   - 启动实例（平台会自动分配 gRPC 端口、生成配置、拉起容器）。

3. **端口映射**：`BrokerPort`（默认 1883）是设备接入端口，
   平台启动容器时按驱动实例的 Docker 参数映射，例如 `-p 1883:1883`。

4. **添加设备**：平台「设备管理」新建设备并绑定该驱动实例，
   设备序列号（SN）就是后面 MQTT 主题里用的标识。

## 二、设备侧怎么接

用任意 MQTT 客户端连接驱动的 broker：

| 参数 | 值 |
| --- | --- |
| 地址 | `tcp://<驱动所在主机>:1883` |
| clientId | 设备序列号（SN） |
| username | 设备序列号（SN） |
| password | 平台「设备管理」里该设备的密钥（未配密钥时放行） |

### 主题约定

| 方向 | 主题 | 报文 |
| --- | --- | --- |
| 上行 | `{TopicRoot}/up/{sn}/properties` | `{"temp": 23.5, "rpm": 1450}` |
| 上行 | `{TopicRoot}/up/{sn}/events` | `{"alarm": {"code": "E01", "msg": "超温"}}` |
| 上行 | `{TopicRoot}/up/{sn}/status` | `{"online": true}` |
| 下行 | `{TopicRoot}/down/{sn}/set` | 平台下发的属性设置 |
| 下行 | `{TopicRoot}/down/{sn}/get` | 平台下发的属性查询 |
| 下行 | `{TopicRoot}/down/{sn}/service` | 平台下发的服务调用 |

### 联调检查清单

- [ ] 设备能连上 broker（clientId/username 用 SN）
- [ ] 往 `iot/up/{sn}/properties` 发一条属性，平台「历史查询」能看到
- [ ] 设备离线后，平台设备列表在超时时间后显示离线
- [ ] 平台下发属性设置，设备在 `iot/down/{sn}/set` 收到

## 三、自定义参数

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `TopicRoot` | `iot` | 主题根 |
| `BrokerPort` | `1883` | 内嵌 broker 监听端口 |
| `AllowAnonymousDevice` | `false` | 预留：是否允许未注册设备连接 |

## 四、本地开发

```bash
go mod tidy
go build ./...
```

驱动依赖平台下发的配置（`/etc/driver/res/configuration.toml`，含平台 gRPC 地址与实例 ID），
**不能脱离平台单独跑**。要本地调试，起一个平台实例后按上面「平台侧怎么用」的流程创建驱动实例即可
（平台会把配置挂进容器）。

启动日志里能看到关键信息：

```
[server] gRPC 服务启动在 0.0.0.0:<平台分配的端口>
[app] 已连接平台 gRPC <平台地址>
[mqtt] mqtt broker listening on 0.0.0.0:1883
[mqtt] 已连接 broker tcp://127.0.0.1:1883
```