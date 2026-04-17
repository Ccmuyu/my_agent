---
name: weather
description: 查询指定城市或用户所在地的当前天气状态。当用户询问天气或天气状况时调用。
parameters:
  type: object
  properties:
    city:
      type: string
      description: "城市名称。如果不提供，则自动通过IP定位获取用户所在城市。"
---

# Weather Query

查询指定城市或用户所在地的当前天气状态。

## 功能说明

- **自动定位**：如果不提供城市参数，自动通过用户IP定位所在城市
- **天气查询**：返回天气现象、温度、湿度、风向、风力等信息

## 实现步骤

1. 如果未提供城市参数，先通过 IP 定位获取城市
2. 使用公共天气 API wttr.in 查询天气
3. 解析返回结果并整理输出

## 注意事项

- 使用 wttr.in 公共 API，无需 API Key
- 返回简洁的天气摘要信息