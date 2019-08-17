STATIC_SRC_DIR := src
STATIC_BIN_DIR := static

FONTS_SRC_DIR := node_modules/@fortawesome/fontawesome-free/webfonts
FONTS_BIN_DIR := $(STATIC_BIN_DIR)/fonts

FONTS_SRC := $(wildcard $(FONTS_SRC_DIR)/fa-solid-*)
FONTS_BIN := $(addprefix $(FONTS_BIN_DIR)/,$(notdir $(FONTS_SRC)))

STATIC := $(addprefix $(STATIC_BIN_DIR)/,index.html style.css site.js logo.png) $(FONTS_BIN)
PACKR := a_main-packr.go

SRC := umverteiler-web.go
BIN := umverteiler-web

.PHONY: all
all: $(BIN)

$(BIN): $(SRC) $(PACKR)
	go build -o $@ $^

$(PACKR): $(SRC) $(STATIC)
	packr

$(STATIC_BIN_DIR)/%.html: $(STATIC_SRC_DIR)/%.slim | $(STATIC_BIN_DIR)
	bundle exec slimrb $^ > $@

$(STATIC_BIN_DIR)/%.css: $(STATIC_SRC_DIR)/%.sass | $(STATIC_BIN_DIR)
	yarn sass -s compressed --no-source-map $^ $@

$(STATIC_BIN_DIR)/%.js: $(STATIC_SRC_DIR)/%.js | $(STATIC_BIN_DIR)
	yarn uglifyjs -c -m -o $@ $^

$(STATIC_BIN_DIR)/%.png: $(STATIC_SRC_DIR)/%.png | $(STATIC_BIN_DIR)
	cp $^ $@

$(FONTS_BIN_DIR)/%: $(FONTS_SRC_DIR)/% | $(FONTS_BIN_DIR)
	cp $^ $@

$(FONTS_BIN_DIR): | $(STATIC_BIN_DIR)
	mkdir $@

$(STATIC_BIN_DIR):
	mkdir $@

.PHONY: clean
clean:
	rm -rf $(STATIC_BIN_DIR) $(PACKR) $(BIN)
