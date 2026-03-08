#
# Makefile for lunar-tools - a collection of lunar-related shell tools
#

# versioning scheme: since this is mostly a linear process if incremental
# but we do not update that often we use year.number as version number
# i.e. 2004.9 2004.10 2004.11 ...
VERSION = 2026.2

bin_PROGS = prog/run-parts
sbin_PROGS = prog/lids prog/luser prog/lnet prog/lservices \
	prog/lmodules prog/clad prog/ltime prog/installkernel
lnet_LIBS = lib/lnet/bootstrap \
            lib/lnet/config_file.lnet.sh \
            lib/lnet/device_config.lnet.sh \
            lib/lnet/devices.lnet.sh \
            lib/lnet/dialog.lnet.sh \
            lib/lnet/menus.lnet.sh \
            lib/lnet/netmasks.lnet.sh \
            lib/lnet/wifi.lnet.sh
DOCS = README.md COPYING
MANPAGES = $(shell ls -1 man/*)

lnet_LIBS_dir = /var/lib/lunar/functions/lnet

all:

.PHONY:
install: .PHONY
	install -d $(DESTDIR)$(lnet_LIBS_dir)
	for LIB in $(lnet_LIBS) ; do \
	    install -m644 $${LIB} $(DESTDIR)$(lnet_LIBS_dir) ; \
	done
	install -d $(DESTDIR)/usr/bin
	for PROGRAM in ${bin_PROGS} ; do \
	    install -m755 $${PROGRAM} $(DESTDIR)/usr/bin/ ; \
	done
	install -d $(DESTDIR)/usr/sbin
	for PROGRAM in ${sbin_PROGS} ; do \
	    install -m755 $${PROGRAM} $(DESTDIR)/usr/sbin/ ; \
	done
	for MANPAGE in ${MANPAGES} ; do \
	    EXT=`echo "$${MANPAGE:(($${#MANPAGE}-1)):1}"` ; \
	    install -d $(DESTDIR)/usr/share/man/man$$EXT ; \
	    install -m644 $${MANPAGE} $(DESTDIR)/usr/share/man/man$$EXT/ ; \
	done
	install -d $(DESTDIR)/usr/share/doc/lunar-tools
	for DOC in ${DOCS} ; do \
		install -m644 $${DOC} $(DESTDIR)/usr/share/doc/lunar-tools/ ; \
	done

dist:
	git archive --format=tar --prefix=lunar-tools-$(VERSION)/ lunar-tools-$(VERSION) | bzip2 > lunar-tools-$(VERSION).tar.bz2

# Go tool variables
GOOS ?= linux
GOARCH ?= amd64
ARCH ?= $(GOARCH)
GO_TOOLS = $(dir $(wildcard tools/*/go.mod))
go_PROGS = tools/llint/llint

COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build-tools: .PHONY
	@for moddir in $(GO_TOOLS); do \
	    tool=$$(basename "$$moddir") ; \
	    echo "Building $$tool for $(GOOS)/$(GOARCH)" ; \
	    cd "$$moddir" && \
	    GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	        -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)" \
	        -o "$$tool" . && \
	    cd "$(CURDIR)" ; \
	done

install-all: install
	install -d $(DESTDIR)/usr/bin
	for PROGRAM in $(go_PROGS) ; do \
	    install -m755 $${PROGRAM} $(DESTDIR)/usr/bin/ ; \
	done

release: build-tools
	$(eval TMPDIR := $(shell mktemp -d))
	git archive --format=tar --prefix=lunar-tools-$(VERSION)/ HEAD | tar -C $(TMPDIR) -xf -
	rm -rf $(TMPDIR)/lunar-tools-$(VERSION)/.github
	find $(TMPDIR)/lunar-tools-$(VERSION)/tools -name '*.go' -o -name 'go.mod' -o -name 'go.sum' -o -name '.gitignore' | xargs rm -f
	@for moddir in $(GO_TOOLS); do \
	    tool=$$(basename "$$moddir") ; \
	    cp "$$moddir$$tool" "$(TMPDIR)/lunar-tools-$(VERSION)/$$moddir" ; \
	done
	tar -C $(TMPDIR) -cJf lunar-tools-$(VERSION)-$(ARCH).tar.xz lunar-tools-$(VERSION)/
	rm -rf $(TMPDIR)
