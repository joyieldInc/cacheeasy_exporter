
GO := go

target = cacheeasy_exporter

all: build

release: format build

build: $(target)

$(target): $(target).go
	$(GO) build $(target).go

format:
	$(GO) fmt $(target).go

clean:
	@rm -rf $(target)
	@echo Done.
