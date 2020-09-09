package main

import (
	"fmt"
	"time"

	"github.com/xuperchain/xuper-sdk-go/account"
	"github.com/xuperchain/xuper-sdk-go/cross_query"
	"github.com/xuperchain/xuper-sdk-go/pb"
)

// define blockchain node and blockchain name
var (
	//node = "14.215.179.74:37101"
	node         = "127.0.0.1:37101"
	bcname       = "xuper"
	address      = "WwLgfAatHyKx2mCJruRaML4oVf7Chzp42"
	pubkey       = `{"Curvname":"P-256","X":59572894642662849351951007648381266067965665107900867144213709334891664628384,"Y":8048742862014975230056503560798576017872466904786606109303178975385452397337}`
	args         = map[string]string{"key": "a"}
	methodName   = "get"
	contractName = "counter"
)

func main() {
	acc, err := account.CreateAccount(1, 1)
	if err != nil {
		fmt.Printf("create account error: %v\n", err)
		return
	}

	queryMeta := &pb.CrossQueryMeta{
		ChainMeta: &pb.CrossChainMeta{
			Type:           "xuper",
			MinEndorsorNum: 1,
		},
		Endorsors: []*pb.CrossEndorsor{
			{
				Address: address,
				PubKey:  pubkey,
				Host:    node,
			},
		},
	}

	crossQueryRequest := &pb.CrossQueryRequest{
		Bcname:      bcname,
		Timestamp:   time.Now().UnixNano(),
		Initiator:   acc.Address,
		AuthRequire: []string{address},
		Request:     cross_query.GenerateInvokeIR(args, methodName, contractName),
	}

	res, err := cross_query.CrossQuery(crossQueryRequest, queryMeta)
	if err != nil {
		fmt.Printf("cross query error: %v\n", err)
	}
	fmt.Printf("cross query res: %v\n", res)
}
