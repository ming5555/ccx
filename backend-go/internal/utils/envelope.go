package utils

import (
	"bytes"
	"encoding/json"
)

// UnwrapGatewayEnvelope 解开部分 OpenAI 兼容网关（如 Cline / OpenRouter 转发层）
// 对非流式响应做的 {data:{...}} 信封包装。
//
// 背景：这类网关会把标准的 chat.completion 对象整体塞进 data 字段，例如
//
//	{"success":true,"data":{"choices":[...],"usage":{...}}}
//
// CCX 按标准 OpenAI 结构读取顶层 choices/usage，遇到这种响应会误判为
// 「上游返回空响应」并触发 failover。本函数仅在上游确实使用信封时解包，
// 其余情况原样返回，对标准上游零副作用。
//
// 判定条件（保守）：
//  1. 顶层是 JSON object 且不含 choices；
//  2. 顶层 data 是 JSON object 且其内部含 choices。
//
// 命中时返回 data 的原始 JSON 字节（不重新序列化，避免形态变化）与 true。
func UnwrapGatewayEnvelope(body []byte) ([]byte, bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) < 2 || trimmed[0] != '{' {
		return body, false
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &top); err != nil {
		return body, false
	}
	// 已经是标准结构（顶层含 choices），不动
	if _, ok := top["choices"]; ok {
		return body, false
	}

	data, ok := top["data"]
	if !ok {
		return body, false
	}
	inner := bytes.TrimSpace(data)
	if len(inner) < 2 || inner[0] != '{' {
		return body, false
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(inner, &dataMap); err != nil {
		return body, false
	}
	if _, ok := dataMap["choices"]; !ok {
		return body, false
	}
	return data, true
}
