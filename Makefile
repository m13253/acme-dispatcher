.PHONY: all clean install uninstall

GOBUILD=go build
GOGET=go get -d -v .
PREFIX=/usr/local

all: acme-dispatcher

clean:
	rm -f acme-dispatcher

install: acme-dispatcher
	install -Dm0755 acme-dispatcher "$(DESTDIR)$(PREFIX)/bin/acme-dispatcher"
	[ -e "$(DESTDIR)/etc/acme-dispatcher.conf" ] || install -Dm0644 acme-dispatcher.conf "$(DESTDIR)/etc/acme-dispatcher.conf"
	$(MAKE) -C systemd install "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/acme-dispatcher"
	$(MAKE) -C systemd uninstall "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"

acme-dispatcher: config.go main.go server.go
	$(GOGET) && $(GOBUILD)
