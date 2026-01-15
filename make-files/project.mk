# project.mk 汇总跨领域的组合目标

.PHONY: all
all: clean mod-tidy build test package ## 完整构建与打包流程
