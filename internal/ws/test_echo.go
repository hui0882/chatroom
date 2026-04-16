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
			data = reverseBytes(data)
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

// reverseBytes 将字节切片中的字符倒序（按字节逆转，适用于 ASCII 文本）。
func reverseBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i, v := range b {
		out[len(b)-1-i] = v
	}
	return out
}
