VERSION=$(shell (head -1 VERSION))

ZIP_NAME				:= mosi-docker-registry

BIN_DIR					:= _bin
BIN_TOOLS_DIR			:= $(BIN_DIR)/tools
ZIP_DIR					:= _zip
DEV_DIR					:= _dev

FLAGS					:= -ldflags "-X 'main.Version=$(VERSION)'"

ROOT_DIR				:= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

BIN_DIR_LINUX			:= $(ROOT_DIR)$(BIN_DIR)/linux/
BIN_TOOLS_DIR_LINUX		:= $(BIN_DIR_LINUX)tools/
BIN_DIR_MACOS			:= $(ROOT_DIR)$(BIN_DIR)/macos/
BIN_TOOLS_DIR_MACOS		:= $(BIN_DIR_MACOS)tools/
BIN_DIR_WINDOWS			:= $(ROOT_DIR)$(BIN_DIR)/windows/
BIN_TOOLS_DIR_WINDOWS	:= $(BIN_DIR_WINDOWS)tools/


ZIP_FILE_LINUX			:= $(ROOT_DIR)$(ZIP_DIR)/$(ZIP_NAME)-$(VERSION)-linux.zip
ZIP_FILE_MACOS			:= $(ROOT_DIR)$(ZIP_DIR)/$(ZIP_NAME)-$(VERSION)-macos.zip
ZIP_FILE_WINDOWS		:= $(ROOT_DIR)$(ZIP_DIR)/$(ZIP_NAME)-$(VERSION)-windows.zip

ZIP_TMP_DIR				:= $(ROOT_DIR)_ziptmpdir


all:				test \
					build \
					zip \

build:				build-linux \
					build-macos \
					build-windows \

build-dev-linux:	build-linux \
					deploy-dev-linux \

build-dev-macos:	build-macos \
					deploy-dev-macos \

build-dev-windows:	build-windows \
					deploy-dev-windows \

zip:				zip-init \
					zip-impl-linux \
					zip-impl-macos \
					zip-impl-windows \
					zip-done \

zip-linux:			zip-init \
					zip-impl-linux \
					zip-done \

zip-macos:			zip-init \
					zip-impl-macos \
					zip-done \

zip-windows:		zip-init \
					zip-impl-windows \
					zip-done \

release:			tag \
					build \
					zip \


test:
	@echo "==> TEST"
	@go test -v ./cmd/...
	@go test -v ./pkg/...

build-linux:
	@echo "==> BUILD  LINUX"
	@rm -Rf $(BIN_DIR_LINUX)
	@GOOS=linux GOARCH=amd64 go build $(FLAGS) -o $(BIN_DIR_LINUX) ./cmd/mosi
	@GOOS=linux GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_LINUX) ./cmd/generate-server-certificate
	@GOOS=linux GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_LINUX) ./cmd/configure-docker-toolbox
	@chmod -R +x $(BIN_DIR_LINUX)

build-macos:
	@echo "==> BUILD  MACOS"
	@rm -Rf $(BIN_DIR_MACOS)
	@GOOS=darwin GOARCH=amd64 go build $(FLAGS) -o $(BIN_DIR_MACOS) ./cmd/mosi
	@GOOS=darwin GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_MACOS) ./cmd/generate-server-certificate
	@GOOS=darwin GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_MACOS) ./cmd/configure-docker-toolbox
	@chmod -R +x $(BIN_DIR_MACOS)

build-windows:
	@echo "==> BUILD  WINDOWS"
	@rm -Rf $(BIN_DIR_WINDOWS)
	@GOOS=windows GOARCH=amd64 go build $(FLAGS) -o $(BIN_DIR_WINDOWS) ./cmd/mosi
	@GOOS=windows GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_WINDOWS) ./cmd/generate-server-certificate
	@GOOS=windows GOARCH=amd64 go build $(FLAGS) -o $(BIN_TOOLS_DIR_WINDOWS) ./cmd/configure-docker-toolbox


deploy-dev-linux:
	@mkdir -p $(DEV_DIR); \
	cp -R $(BIN_DIR_LINUX)* $(DEV_DIR); \

deploy-dev-macos:
	@mkdir -p $(DEV_DIR); \
	cp -R $(BIN_DIR_MACOS)* $(DEV_DIR); \

deploy-dev-windows:
	@mkdir -p $(DEV_DIR); \
	cp -R $(BIN_DIR_WINDOWS)* $(DEV_DIR); \


zip-init:
	@mkdir -p $(ZIP_DIR); \
	mkdir -p $(ZIP_TMP_DIR)/$(ZIP_NAME); \
	mkdir -p $(ZIP_TMP_DIR)/$(ZIP_NAME)/scripts; \
	cp scripts/*.sh $(ZIP_TMP_DIR)/$(ZIP_NAME)/scripts/; \
	mkdir -p $(ZIP_TMP_DIR)/$(ZIP_NAME)/certs; \
	cp setup/certs/*.* $(ZIP_TMP_DIR)/$(ZIP_NAME)/certs; \

zip-done:
	@rm -R $(ZIP_TMP_DIR)

zip-impl-linux:
	@echo "==> ZIP LINUX   $(ZIP_FILE_LINUX)"
	@rm -f $(ZIP_TMP_DIR)/$(ZIP_NAME)/* 2> /dev/null; \
	rm -Rf $(ZIP_TMP_DIR)/$(ZIP_NAME)/tools 2> /dev/null; \
	cp -R $(BIN_DIR_LINUX)/* $(ZIP_TMP_DIR)/$(ZIP_NAME); \
	cd $(ZIP_TMP_DIR); \
	rm -f $(ZIP_FILE_LINUX); \
	zip -q -r $(ZIP_FILE_LINUX) $(ZIP_NAME); \

zip-impl-macos:
	@echo "==> ZIP MACOS   $(ZIP_FILE_MACOS)"
	@rm -f $(ZIP_TMP_DIR)/$(ZIP_NAME)/* 2> /dev/null; \
	rm -Rf $(ZIP_TMP_DIR)/$(ZIP_NAME)/tools 2> /dev/null; \
	cp -R $(BIN_DIR_MACOS)/* $(ZIP_TMP_DIR)/$(ZIP_NAME); \
	cd $(ZIP_TMP_DIR); \
	rm -f $(ZIP_FILE_MACOS); \
	zip -q -r $(ZIP_FILE_MACOS) $(ZIP_NAME); \

zip-impl-windows:
	@echo "==> ZIP WINDOWS $(ZIP_FILE_WINDOWS)"
	@rm -f $(ZIP_TMP_DIR)/$(ZIP_NAME)/* 2> /dev/null; \
	rm -Rf $(ZIP_TMP_DIR)/$(ZIP_NAME)/tools 2> /dev/null; \
	cp -R $(BIN_DIR_WINDOWS)/* $(ZIP_TMP_DIR)/$(ZIP_NAME); \
	cd $(ZIP_TMP_DIR); \
	rm -f $(ZIP_FILE_WINDOWS); \
	zip -q -r $(ZIP_FILE_WINDOWS) $(ZIP_NAME); \
