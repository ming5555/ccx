package utils

import (
	"encoding/json"
	"testing"
)

func TestUnwrapGatewayEnvelope(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantWrap bool
		// wantEqual 为 true 时要求返回体与输入完全一致（未命中解包）
		wantEqual bool
		// wantChoices 为 true 时要求解包后的 body 顶层含非空 choices
		wantChoices bool
	}{
		{
			name:      "标准OpenAI响应保持原样",
			body:      `{"id":"x","choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":3}}`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:        "data信封解包",
			body:        `{"data":{"id":"x","choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":3}}}`,
			wantWrap:    true,
			wantChoices: true,
		},
		{
			name:        "success加data信封解包",
			body:        `{"success":true,"data":{"object":"chat.completion","choices":[{"message":{"content":"ok"}}]}}`,
			wantWrap:    true,
			wantChoices: true,
		},
		{
			name:      "data内无choices不解包",
			body:      `{"success":true,"data":{"object":"chat.completion","id":"x"}}`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "data为非对象不解包",
			body:      `{"data":"oops"}`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "顶层已有choices时即使有data也不解包",
			body:      `{"choices":[{"message":{"content":"ok"}}],"data":{"foo":1}}`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "非法JSON不解包",
			body:      `{"data":{"choices":[`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "非对象JSON不解包",
			body:      `[1,2,3]`,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "空body不解包",
			body:      ``,
			wantWrap:  false,
			wantEqual: true,
		},
		{
			name:      "普通错误体不解包",
			body:      `{"error":"invalid model format","success":false}`,
			wantWrap:  false,
			wantEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, wrapped := UnwrapGatewayEnvelope([]byte(tt.body))
			if wrapped != tt.wantWrap {
				t.Fatalf("wrapped = %v, want %v", wrapped, tt.wantWrap)
			}
			if tt.wantEqual && string(got) != tt.body {
				t.Fatalf("未命中解包时应原样返回, got %q, want %q", string(got), tt.body)
			}
			if tt.wantChoices {
				var m map[string]json.RawMessage
				if err := json.Unmarshal(got, &m); err != nil {
					t.Fatalf("解包后 JSON 非法: %v", err)
				}
				choices, ok := m["choices"]
				if !ok || len(choices) == 0 || string(choices) == "null" {
					t.Fatalf("解包后顶层应含非空 choices, got %q", string(got))
				}
				if _, ok := m["usage"]; ok {
					// usage 若在信封内，解包后应位于顶层（此处仅断言可解析）
				}
			}
		})
	}
}

// TestUnwrapGatewayEnvelope_UsageHoisted 验证信封内的 usage 会随解包提升到顶层，
// 保证计费/统计路径能读到 usage。
func TestUnwrapGatewayEnvelope_UsageHoisted(t *testing.T) {
	body := []byte(`{"success":true,"data":{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":7,"completion_tokens":2}}}`)
	got, ok := UnwrapGatewayEnvelope(body)
	if !ok {
		t.Fatal("应当命中解包")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("解包后 JSON 非法: %v", err)
	}
	usageRaw, ok := m["usage"]
	if !ok {
		t.Fatalf("解包后应在顶层含 usage, got %s", string(got))
	}
	var usage map[string]float64
	if err := json.Unmarshal(usageRaw, &usage); err != nil {
		t.Fatalf("usage 解析失败: %v", err)
	}
	if usage["prompt_tokens"] != float64(7) {
		t.Fatalf("usage.prompt_tokens = %v, want 7", usage["prompt_tokens"])
	}
}
