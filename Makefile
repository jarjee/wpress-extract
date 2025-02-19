BIN_NAME = wpress-extract
 OUTPUT_DIR = bin

 .PHONY: build clean

 build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build -o $(OUTPUT_DIR)/$(BIN_NAME) main.go

 clean:
	@rm -rf $(OUTPUT_DIR)
