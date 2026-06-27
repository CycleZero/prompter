BINARY_NAME := app
BINARY_PATH := ./bin/${BINARY_NAME}
MAIN_FILE_DIR := ./

# wire 依赖注入代码生成
wire:
	wire ${MAIN_FILE_DIR}

# 前端构建
web:
	cd web && pnpm install && pnpm run build

# 编译（含前端）
build: web
	go build -o ${BINARY_PATH} ${MAIN_FILE_DIR}

# 一键重新构建（wire + build）
rebuild: wire build

swag:
	swag init

# 编译至 Linux AMD64 平台（含前端）
build-linux: web
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY_PATH} ${MAIN_FILE_DIR}

# 运行
run:
	go run .

# 安装依赖
tidy:
	go mod tidy
