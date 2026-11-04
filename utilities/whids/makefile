TEST=$(GOPATH)/test
MAIN_BASEN_SRC=whids
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
	$(MAKE) installer
	$(MAKE) sysmon
	$(MAKE) dlrules

test:
	$(MAKE) clean
	$(MAKE) init
	$(MAKE) buildversion
	$(MAKE) compile
	$(MAKE) installer
	$(MAKE) sysmon
	$(MAKE) testrules
	cp -r $(RELEASE)/.. $(TEST)

init:
	mkdir -p $(RELEASE)
	mkdir -p $(RELEASE)/windows

install:
	go install $(OPTS) ./

compile: windows

windows:
	GOARCH=386 GOOS=windows go build $(OPTS) -o $(RELEASE)/windows/$(MAIN_BASEN_SRC)-v$(VERSION)-386.exe ./
	GOARCH=amd64 GOOS=windows go build $(OPTS) -o $(RELEASE)/windows/$(MAIN_BASEN_SRC)-v$(VERSION)-amd64.exe ./
	#cp -r conf $(RELEASE)/windows
	cd $(RELEASE)/windows; find -type f | xargs -I "{}" shasum -a 256 {} >> sha256.txt
	#cd $(RELEASE)/windows; tar -cvzf ../$(MAIN_BASEN_SRC)-windows-$(VERSION).tar.gz *

buildversion:
	@echo "version $(VERSION) commit $(COMMITID) (injected via -ldflags, see version.go)"

installer:
	sed -E -e "s/set VERSION=.*?/set VERSION=$(VERSION)/" -e "s/set COMMITID=.*?/set COMMITID=$(COMMITID)/" manage.bat > $(RELEASE)/windows/manage.bat

sysmon:
	#mv $(RELEASE)/../sysmon $(RELEASE)/windows/

dlrules:
	mkdir $(RELEASE)/windows/rules/
	curl https://raw.githubusercontent.com/0xrawsec/gene-rules/master/compiled.gen > $(RELEASE)/windows/rules/compiled.gen
	curl https://raw.githubusercontent.com/0xrawsec/gene-rules/master/compiled.gen.sha256 > $(RELEASE)/windows/rules/compiled.gen.sha256

testrules:
	mkdir $(RELEASE)/windows/rules/
	cp $(TEST_RULES)/compiled.gen $(RELEASE)/windows/rules/compiled.gen
	cp $(TEST_RULES)/compiled.gen.sha256 $(RELEASE)/windows/rules/compiled.gen.sha256

clean:
	rm -rf $(RELEASE)/*
