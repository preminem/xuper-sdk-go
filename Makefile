ifeq ($(OS),Windows_NT)
	PLATFORM="Windows"
else
	ifeq ($(shell uname),Darwin)
		PLATFORM="MacOS"
	else
	    PLATFORM="Linux"
	endif
endif

all: build
export GO111MODULE=on
export GOFLAGS=-mod=vendor
export OUTPUT=./output

build:
	PLATFORM=$(PLATFORM) ./build.sh

test:
	go test github.com/xuperchain/xuper-sdk-go/account
	go test github.com/xuperchain/xuper-sdk-go/transfer
	go test github.com/xuperchain/xuper-sdk-go/common
	go test github.com/xuperchain/xuper-sdk-go/contract_account
	go test github.com/xuperchain/xuper-sdk-go/contract
	go test github.com/xuperchain/xuper-sdk-go/event
	go test github.com/xuperchain/xuper-sdk-go/cross_query
	go test github.com/xuperchain/xuper-sdk-go/xchain

clean:
	rm -rf main
	rm -rf sample

.PHONY: all test clean
