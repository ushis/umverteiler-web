FONTS := $(wildcard node_modules/@fortawesome/fontawesome-free/webfonts/fa-solid-*)
STATIC := static/index.html static/style.css static/site.js static/logo.png static/fonts
PACKR := a_main-packr.go

SRC := umverteiler-web.go
BIN := umverteiler-web

.PHONY: all
all: $(BIN)

$(BIN): $(SRC) $(PACKR)
	go build -o $@ $^

$(PACKR): $(SRC) $(STATIC)
	packr

.PHONY: static
static: $(STATIC)

static/index.html: src/index.slim
	bundle exec slimrb $^ > $@

static/style.css: src/style.sass
	bundle exec sass -t compressed $^ > $@

static/site.js: src/site.js
	./node_modules/.bin/uglifyjs -c -m -o $@ $^

static/logo.png: src/logo.png
	cp $^ $@

static/fonts: $(FONTS)
	mkdir -p $@
	cp $^ $@

.PHONY: clean
clean:
	rm -rf $(STATIC) $(PACKR) $(BIN)
