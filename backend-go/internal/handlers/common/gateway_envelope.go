package common

import (
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

// UnwrapGatewayEnvelope 解开上游网关信封并记录日志，详见 utils.UnwrapGatewayEnvelope。
// tag 用于日志前缀（如 Messages / Chat / Responses / Gemini）。
func UnwrapGatewayEnvelope(c *gin.Context, tag string, body []byte) []byte {
	unwrapped, ok := utils.UnwrapGatewayEnvelope(body)
	if !ok {
		return body
	}
	RequestLogf(c, "[%s-Gateway-Envelope] 检测到网关信封响应，已解包 {data:{...}}（上游将 choices/usage 包在 data 内）", tag)
	return unwrapped
}
