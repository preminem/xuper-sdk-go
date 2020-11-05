package cross_query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	crypto_client "github.com/xuperchain/xuper-sdk-go/crypto"
	"github.com/xuperchain/xuper-sdk-go/pb"
)

type TxQueryInfo struct {
	Request  *pb.TxStatus        `json:"request"`
	Response *pb.Transaction     `json:"response"`
	Signs    []*pb.SignatureInfo `json:"signs"`
}

type txQueryRes struct {
	queryRes *pb.Transaction
	signs    *pb.SignatureInfo
}

// TxQuery query tx from otherchain
func TxQuery(txQueryRequest *pb.TxStatus, queryMeta *pb.CrossQueryMeta) (*TxQueryInfo, error) {
	log.Printf("Receive TxQuery,txQueryRequest:%v, queryMeta:%v\n", txQueryRequest, queryMeta)
	if !isQueryMetaValid(queryMeta) {
		return nil, fmt.Errorf("isQueryParamValid check failed")
	}
	// Call endorsor for responce
	queryInfo, err := txQueryFromEndorsor(txQueryRequest, queryMeta)
	if err != nil {
		log.Printf("txQueryFromEndorsor error:%v", err)
		return nil, err
	}

	// 验证背书规则、参数有效性、时间戳有效性
	// 验证request、签名等信息
	for !IsTxQueryValid(txQueryRequest, queryMeta, queryInfo) {
		return nil, fmt.Errorf("IsTxQueryValid check failed")
	}
	return queryInfo, nil
}

// txQueryFromEndorsor will query tx from endorsor
func txQueryFromEndorsor(
	txQueryRequest *pb.TxStatus,
	queryMeta *pb.CrossQueryMeta) (*TxQueryInfo, error) {

	reqData, err := json.Marshal(txQueryRequest)
	if err != nil {
		return nil, err
	}

	req := &pb.EndorserRequest{
		RequestName: "TxQuery",
		BcName:      txQueryRequest.GetBcname(),
		RequestData: reqData,
	}

	res, signs, err := txQueryWithGroup(req, queryMeta)
	if err != nil {
		return nil, err
	}
	return &TxQueryInfo{
		Request:  txQueryRequest,
		Response: res,
		Signs:    signs,
	}, nil
}

func txQueryWithGroup(req *pb.EndorserRequest, queryMeta *pb.CrossQueryMeta) (*pb.Transaction, []*pb.SignatureInfo, error) {
	wg := sync.WaitGroup{}
	msgChan := make(chan *txQueryRes, len(queryMeta.GetEndorsors()))

	for idx := range queryMeta.GetEndorsors() {
		wg.Add(1)
		go func(req *pb.EndorserRequest, ce *pb.CrossEndorsor) {
			defer wg.Done()
			res, err := txQuery(req, ce)
			if err != nil {
				return
			}
			msgChan <- res
		}(req, queryMeta.GetEndorsors()[idx])
	}
	wg.Wait()
	// 处理所有请求结果
	signs := []*pb.SignatureInfo{}
	var conRes *pb.Transaction
	lenCh := len(msgChan)
	if lenCh <= 0 {
		return nil, nil, errors.New("txQueryWithGroup res is nil")
	}

	breakFlag := 0
	for r := range msgChan {
		if breakFlag == 0 {
			conRes = r.queryRes
		} else {
			if !isMsgEqual(conRes, r.queryRes) {
				return conRes, signs, errors.New("txQueryWithGroup ContractResponse different")
			}
		}
		signs = append(signs, r.signs)
		if breakFlag >= lenCh-1 {
			break
		}
		breakFlag++
	}
	return conRes, signs, nil
}

func txQuery(req *pb.EndorserRequest, ce *pb.CrossEndorsor) (*txQueryRes, error) {
	ctx, _ := context.WithTimeout(context.TODO(), endorserTimeOut)
	conn, err := newEndorsorConn(ce.GetHost())
	if err != nil {
		log.Printf("txQuery NewEndorsorClient error:%v\n", err)
		return nil, err
	}
	defer conn.Close()
	cli := pb.NewXendorserClient(conn)
	endorsorRes, err := cli.EndorserCall(ctx, req)
	if err != nil {
		log.Printf("txQuery EndorserCall error:%v\n", err)
		return nil, err
	}
	res := &pb.Transaction{}
	err = json.Unmarshal(endorsorRes.GetResponseData(), res)
	if err != nil {
		log.Printf("txQuery Unmarshal error:%v\n", err)
		return nil, err
	}
	txQueryRes := &txQueryRes{
		queryRes: res,
		signs:    endorsorRes.GetEndorserSign(),
	}
	return txQueryRes, nil
}

// 验证CrossQuery背书信息
func IsTxQueryValid(
	request *pb.TxStatus,
	queryMeta *pb.CrossQueryMeta,
	queryInfo *TxQueryInfo) bool {
	// check req from client and req from preexec
	if !isMsgEqual(request, queryInfo.Request) {
		log.Println("IsTxQueryValid isMsgEqual not equal")
		return false
	}

	// check endorsor info
	signs, ok := isEndorsorInfoValid(queryMeta, queryInfo.Signs)
	if !ok {
		log.Println("isTxSignValid not ok")
		return false
	}

	// check endorsor sign
	if !isTxSignValid(signs, queryInfo) {
		log.Println("isTxSignValid not ok")
		return false
	}
	return true
}

func isTxSignValid(signsValid []*pb.SignatureInfo, queryInfo *TxQueryInfo) bool {
	reqData, err := json.Marshal(queryInfo.Request)
	if err != nil {
		log.Printf("Marshal Request failed:%v\n", err)
		return false
	}
	resData, err := json.Marshal(queryInfo.Response)
	if err != nil {
		log.Printf("Marshal Response failed:%v\n", err)
		return false
	}
	cryptoClient := crypto_client.GetCryptoClient()
	data := append(reqData[:], resData[:]...)
	digest := usingSha256(data)
	for idx := range signsValid {
		pk, err := cryptoClient.GetEcdsaPublicKeyFromJsonStr(signsValid[idx].GetPublicKey())
		if err != nil {
			log.Println("GetEcdsaPublicKeyFromJSON failed")
			return false
		}
		ok, err := cryptoClient.VerifyECDSA(pk, signsValid[idx].GetSign(), digest)
		if !ok || err != nil {
			log.Printf("VerifyECDSA failed:%v\n", err)
			return false
		}
	}
	return true
}
