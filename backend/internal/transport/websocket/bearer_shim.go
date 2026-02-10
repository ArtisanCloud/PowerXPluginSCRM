package websocket

import (
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

func b64urlDecode(s string) (string, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// BearerShim promotes query/subprotocol token to Authorization.
func BearerShim() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			if auth := c.Query("authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				c.Request.Header.Set("Authorization", auth)
			}

			if c.GetHeader("Authorization") == "" {
				vals := c.Request.Header.Values("Sec-WebSocket-Protocol")
				if len(vals) == 0 {
					if v := c.GetHeader("Sec-WebSocket-Protocol"); v != "" {
						vals = []string{v}
					}
				}
				for _, v := range vals {
					for _, p := range strings.Split(v, ",") {
						pp := strings.TrimSpace(p)
						if strings.HasPrefix(strings.ToLower(pp), "bearer.") {
							raw := pp[strings.IndexByte(pp, '.')+1:]
							if tok, err := b64urlDecode(raw); err == nil && tok != "" {
								c.Request.Header.Set("Authorization", "Bearer "+tok)
								c.Writer.Header().Set("Sec-WebSocket-Protocol", pp)
								goto NEXT
							}
						}
					}
				}
			}
		}
	NEXT:
		c.Next()
	}
}
