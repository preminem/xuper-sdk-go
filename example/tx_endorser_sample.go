package main

import (
	"encoding/hex"
	"fmt"

	"github.com/xuperchain/xuper-sdk-go/cross_query"
	"github.com/xuperchain/xuper-sdk-go/pb"
)

// define blockchain node and blockchain name
var (
	//node = "14.215.179.74:37101"
	node1    = "127.0.0.1:37101"
	bcname1  = "xuper"
	txid     = "cf20b28962e899ba81263a7c5415b55154f5e679f171638b96c79fb9867f673f"
	address1 = "WwLgfAatHyKx2mCJruRaML4oVf7Chzp42"
	pubkey1  = `{"Curvname":"P-256","X":59572894642662849351951007648381266067965665107900867144213709334891664628384,"Y":8048742862014975230056503560798576017872466904786606109303178975385452397337}`
)

func main() {
	txidByte, err := hex.DecodeString(txid)
	if err != nil {
		fmt.Println("decode hex txid error", "err", err.Error())
		return
	}

	request := &pb.TxStatus{
		Bcname: bcname1,
		Txid:   txidByte,
	}

	queryMeta := &pb.CrossQueryMeta{
		ChainMeta: &pb.CrossChainMeta{
			Type:           "xuper",
			MinEndorsorNum: 1,
		},
		Endorsors: []*pb.CrossEndorsor{
			{
				Address: address1,
				PubKey:  pubkey1,
				Host:    node1,
			},
		},
	}

	res, err := cross_query.TxQuery(request, queryMeta)
	if err != nil {
		fmt.Printf("tx query error: %v\n", err)
		return
	}
	fmt.Printf("tx query res: %v\n", res)
}
