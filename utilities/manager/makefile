TEST=$(GOPATH)/test
MAIN_BASEN_SRC=whids-man
RELEASE=$(GOPATH)/release/$(MAIN_BASEN_SRC)
VERSION?=$(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
  VERSION=dev
endif
COMMITID=$(shell git rev-parse HEAD)

# Build metadata is injected at link time (see version.go)
LDVARS=-X main.version=$(VERSION) -X main.commitID=$(COMMITID)

# Strips symbols and dwarf to make binary smaller
OPTS=-ldflags "-s -w $(LDVARS)" -trimpath
ifdef DEBUG
	OPTS=-ldflags "$(LDVARS)"
endif

all:
	$(MAKE) clean
	$(MAKE) init
	$(MAKE) buildversion
	$(MAKE) compile

test: all
	cp -r $(RELEASE) $(TEST)


init:
	mkdir -p $(RELEASE)
	mkdir -p $(RELEASE)/windows
	mkdir -p $(RELEASE)/linux
	mkdir -p $(RELEASE)/darwin

install:
	go install $(OPTS) ./

compile:
	$(MAKE) windows
	$(MAKE) linux
	$(MAKE) darwin

windows:
	GOARCH=386 GOOS=windows go build $(OPTS) -o $(RELEASE)/windows/$(MAIN_BASEN_SRC)-v$(VERSION)-386.exe ./
	GOARCH=amd64 GOOS=windows go build $(OPTS) -o $(RELEASE)/windows/$(MAIN_BASEN_SRC)-v$(VERSION)-amd64.exe ./
	cd $(RELEASE)/windows; shasum -a 256 * > sha256.txt
	#cd $(RELEASE)/windows; tar -cvzf ../$(MAIN_BASEN_SRC)-windows-$(VERSION).tar.gz *

linux:
	GOARCH=386 GOOS=linux go build $(OPTS) -o $(RELEASE)/linux/$(MAIN_BASEN_SRC)-v$(VERSION)-386 ./
	GOARCH=amd64 GOOS=linux go build $(OPTS) -o $(RELEASE)/linux/$(MAIN_BASEN_SRC)-v$(VERSION)-amd64 ./
	cd $(RELEASE)/linux; shasum -a 256 * > sha256.txt
	#cd $(RELEASE)/linux; tar -cvzf ../$(MAIN_BASEN_SRC)-linux-$(VERSION).tar.gz *

darwin:
	#GOARCH=386 GOOS=darwin go build $(OPTS) -o $(RELEASE)/darwin/$(MAIN_BASEN_SRC)-v$(VERSION)-386 ./
	GOARCH=amd64 GOOS=darwin go build $(OPTS) -o $(RELEASE)/darwin/$(MAIN_BASEN_SRC)-v$(VERSION)-amd64 ./
	cd $(RELEASE)/darwin; shasum -a 256 * > sha256.txt
	#cd $(RELEASE)/darwin; tar -cvzf ../$(MAIN_BASEN_SRC)-darwin-$(VERSION).tar.gz *

buildversion:
	@echo "version $(VERSION) commit $(COMMITID) (injected via -ldflags, see version.go)"

clean:
	rm -rf $(RELEASE)/*
