FONTS := $(wildcard node_modules/@fortawesome/fontawesome-free/webfonts/fa-solid-*)

.PHONY: all
all: umverteiler-web

umverteiler-web: umverteiler-web.go a_main-packr.go
	go build -v -a -o $@ $^

a_main-packr.go: static
	packr

.PHONY: static
static: static/index.html static/style.css static/site.js static/logo.png static/fonts

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
