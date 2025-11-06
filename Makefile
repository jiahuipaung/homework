MODULES = stash
RUN_MODULES = fc-stash

ROOT=$(shell pwd -P)
GIT_COMMIT=$(shell git --work-tree ${ROOT} rev-parse --short HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_TIME=$(shell date +"%Y-%m-%d/%H:%M:%S")
BUILD_VERSION="$(BUILD_TIME)-$(GIT_BRANCH)-$(GIT_COMMIT)"
LDFLAGS:="-w -s -X 'github.com/flashcatcloud/go-pkg/srv.VERSION=$(BUILD_VERSION)'"

all: clean build

clean:
	@rm -rf output

fmt: genv
	gofmt -l -w -s ./app ./handler ./input ./sink ./filter ./extract
	go mod tidy -compat=1.17

lint: genv
# golangci-lint run app/... logic/... model/... structs/... config/... --timeout 5m -j 4

check:
	@grep 'json:' -r ./app/  -r ./input/ -r ./sink/ -r ./filter/ -r ./extract/ -r --exclude '*.swp' | grep -v 'json:"' | awk '{print "json tag error: "; print $1; exit 2}'
	@grep 'bson:' -r ./app/  -r ./input/ -r ./sink/ -r ./filter/ -r ./extract/ -r --exclude '*.swp' | grep -v 'bson:"' | awk '{print "bson tag error: "; print $1; exit 2}'
	@grep '\. "' -r ./app/  -r ./input/ -r ./sink/ -r ./filter/ -r ./extract/ -r --exclude '*.swp' --exclude '*_test.go' | awk '{print "do not use dot import"; print $1; exit 2}'

genv:
	export GOPROXY=https://goproxy.cn,direct
	export GOPRIVATE=github.com/flashcatcloud
	export GOSUMDB=off
	export GO111MODULE=on

build: genv fmt check lint $(MODULES)
	@echo "Build completed"

build-linux-amd: genv fmt check lint $(addprefix linux-amd64-,$(MODULES))
	@echo "Build completed for linux amd64"

build-linux-arm: genv fmt check lint $(addprefix linux-arm64-,$(MODULES))
	@echo "Build completed for linux arm64"

$(MODULES):
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -trimpath -tags=jsoniter -v -o output/bin/fc-$@ github.com/flashcatcloud/fc-stash/app/$@
	cp -r etc scripts output/
	mkdir -p output/logs/

linux-arm64-%:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags $(LDFLAGS) -trimpath -tags=jsoniter -v -o output_linux_arm64/bin/fc-$* github.com/flashcatcloud/fc-stash/app/$*
	cp -r etc scripts output_linux_arm64/
	mkdir -p output_linux_arm64/logs/

linux-amd64-%:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -trimpath -tags=jsoniter -v -o output_linux_amd64/bin/fc-$* github.com/flashcatcloud/fc-stash/app/$*
	cp -r etc scripts output_linux_amd64/
	mkdir -p output_linux_amd64/logs/

$(RUN_MODULES):
	@./output/bin/$@ -c ./output/etc/ -l ./output/logs/ >> ./output/logs/run-$@.log 2>&1 &

dev: $(MODULES)
	@#TODO reset etc file for dev

run: $(RUN_MODULES)
	@ps aux | grep "fc-" | grep -v grep

stop:
	-$(foreach var,$(RUN_MODULES), killall -9 "$(var)";)

up: all stop run