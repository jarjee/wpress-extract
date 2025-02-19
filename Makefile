BIN_NAME = wpress-extract
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ifeq ($(GOOS),windows)
  BIN_NAME := $(BIN_NAME).exe
endif
OUTPUT_DIR = bin
OUTPUT_PATH = $(OUTPUT_DIR)/wpress-extract-$(GOOS)-$(GOARCH)
ifeq ($(GOOS),windows)
  OUTPUT_PATH := $(OUTPUT_PATH).exe
endif

.PHONY: build clean

build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(OUTPUT_PATH) main.go

clean:
	@rm -rf $(OUTPUT_DIR)
