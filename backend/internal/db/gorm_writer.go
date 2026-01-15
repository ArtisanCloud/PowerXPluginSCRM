// internal/db/gorm_writer.go
package db

import (
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"

	gormLogger "gorm.io/gorm/logger"
)

// 满足 gorm.io/gorm/logger.Writer 接口（只有一个 Printf 方法）
type gormWriter struct{}

func (w *gormWriter) Printf(format string, args ...interface{}) {
	// 优先走你项目里的全局 logger
	// - 如果是 logrus.Logger：它有 Printf，等价于 Info 级别
	// - 如果你的 logger 暴露了 Infof/Debugf，也可以换成 appLogger.Infof(...)
	if logger.Logger != nil {
		logger.Logger.Printf(format, args...)
		return
	}
	// 兜底：标准输出
	fmt.Printf(format, args...)
}

// NewGormWriter 返回 GORM 需要的 Writer 适配器
func NewGormWriter() gormLogger.Writer {
	return &gormWriter{}
}
