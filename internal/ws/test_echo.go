// Package ws 提供 /websocket_test 测试接口。
// 该接口无需登录，建立连接后将每条接收到的文本消息倒序后返回给客户端。
package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/logger"
)

var testUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// TestEchoHandler 处理 /websocket_test 连接。
// 无需鉴权，将收到的文本消息倒序后原路返回。
func TestEchoHandler(c *gin.Context) {
	conn, err := testUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("ws_test upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	logger.L().Info("ws_test client connected",
		zap.String("remote", c.Request.RemoteAddr),
	)

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.L().Warn("ws_test read error", zap.Error(err))
			}
			break
		}

		if msgType == websocket.TextMessage {
			data = []byte(reverseRunes(string(data)))
		}

		if err := conn.WriteMessage(msgType, data); err != nil {
			logger.L().Warn("ws_test write error", zap.Error(err))
			break
		}
	}

	logger.L().Info("ws_test client disconnected",
		zap.String("remote", c.Request.RemoteAddr),
	)
}

// reverseRunes 将字符串按 Unicode 码点（rune）倒序，正确处理中文等多字节字符。
func reverseRunes(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
