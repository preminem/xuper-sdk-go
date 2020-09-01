package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xuperchain/xuper-sdk-go/event"
)

// define blockchain node host, blockchain name and event consumer buffer size
var (
	//host = "14.215.179.74:37101"
	host                         = "127.0.0.1:37101"
	blockchainName               = "xuper"
	eventConsumerBufferSize uint = 100
)

func main() {
	watcher, err := event.InitWatcher(host, eventConsumerBufferSize)
	if err != nil {
		log.Printf("Init event watcher err: %v", err)
		return
	}

	blockFilter, err := event.NewBlockFilter(blockchainName, event.WithExcludeTx(true))
	if err != nil {
		log.Printf("Failed to new block filter: %v", err)
		return
	}

	reg, err := watcher.RegisterBlockEvent(blockFilter, false)
	if err != nil {
		log.Printf("Failed to register block event: %v", err)
		return
	}
	defer reg.Unregister()
	var buf []byte
	go func() {
		for {
			select {
			case block, ok := <-reg.FilteredBlockChan:
				if !ok {
					return
				}
				buf, _ = json.MarshalIndent(block, "", "  ")
				fmt.Println(string(buf))
			default:
			}
		}
	}()
}
