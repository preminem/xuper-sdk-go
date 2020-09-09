package cross_query

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	crypto_client "github.com/xuperchain/xuper-sdk-go/crypto"
	"github.com/xuperchain/xuper-sdk-go/pb"
)

const (
	endorserTimeOut = 6 * time.Second
)

type queryRes struct {
	queryRes *pb.CrossQueryResponse
	signs    *pb.SignatureInfo
}

// newEndorsorConn return EndorsorClient
func newEndorsorConn(addr string) (*grpc.ClientConn, error) {
	conn := &grpc.ClientConn{}
	options := append([]grpc.DialOption{}, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, options...)
	if err != nil {
		return nil, errors.New("New grpcs conn error")
	}
	return conn, nil
}

// CrossQuery query contract from otherchain
func CrossQuery(crossQueryRequest *pb.CrossQueryRequest, queryMeta *pb.CrossQueryMeta) (*pb.CrossQueryInfo, error) {
	log.Printf("Receive CrossQuery,crossQueryRequest:%v, queryMeta:%v\n", crossQueryRequest, queryMeta)
	if !isQueryMetaValid(queryMeta) {
		return nil, fmt.Errorf("isQueryParamValid check failed")
	}
	// Call endorsor for responce
	queryInfo, err := crossQueryFromEndorsor(crossQueryRequest, queryMeta)
	if err != nil {
		log.Printf("crossQueryFromEndorsor error:%v", err)
		return nil, err
	}

	// 验证背书规则、参数有效性、时间戳有效性
	// 验证request、签名等信息
	for !IsCossQueryValid(crossQueryRequest, queryMeta, queryInfo) {
		return nil, fmt.Errorf("isCossQueryValid check failed")
	}
	return queryInfo, nil
}

// isQueryParamValid 验证 query meta 背书策略是否有效
func isQueryMetaValid(queryMeta *pb.CrossQueryMeta) bool {
	return len(queryMeta.GetEndorsors()) >= int(queryMeta.GetChainMeta().GetMinEndorsorNum())
}

// crossQueryFromEndorsor will query cross from endorsor
func crossQueryFromEndorsor(
	crossQueryRequest *pb.CrossQueryRequest,
	queryMeta *pb.CrossQueryMeta) (*pb.CrossQueryInfo, error) {

	reqData, err := json.Marshal(crossQueryRequest)
	if err != nil {
		return nil, err
	}

	req := &pb.EndorserRequest{
		RequestName: "CrossQueryPreExec",
		BcName:      crossQueryRequest.GetBcname(),
		RequestData: reqData,
	}

	res, signs, err := endorsorQueryWithGroup(req, queryMeta)
	if err != nil {
		return nil, err
	}
	return &pb.CrossQueryInfo{
		Request:  crossQueryRequest,
		Response: res,
		Signs:    signs,
	}, nil
}

func endorsorQueryWithGroup(req *pb.EndorserRequest, queryMeta *pb.CrossQueryMeta) (*pb.CrossQueryResponse, []*pb.SignatureInfo, error) {
	wg := sync.WaitGroup{}
	msgChan := make(chan *queryRes, len(queryMeta.GetEndorsors()))

	for idx := range queryMeta.GetEndorsors() {
		wg.Add(1)
		go func(req *pb.EndorserRequest, ce *pb.CrossEndorsor) {
			defer wg.Done()
			res, err := endorsorQuery(req, ce)
			if err != nil {
				return
			}
			msgChan <- res
		}(req, queryMeta.GetEndorsors()[idx])
	}
	wg.Wait()
	// 处理所有请求结果
	signs := []*pb.SignatureInfo{}
	var conRes *pb.CrossQueryResponse
	lenCh := len(msgChan)
	if lenCh <= 0 {
		return nil, nil, errors.New("endorsorQueryWithGroup res is nil")
	}

	breakFlag := 0
	for r := range msgChan {
		if breakFlag == 0 {
			conRes = r.queryRes
		} else {
			if !isCrossQueryResponseEqual(conRes, r.queryRes) {
				return conRes, signs, errors.New("endorsorQueryWithGroup ContractResponse different")
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

func endorsorQuery(req *pb.EndorserRequest, ce *pb.CrossEndorsor) (*queryRes, error) {
	ctx, _ := context.WithTimeout(context.TODO(), endorserTimeOut)
	conn, err := newEndorsorConn(ce.GetHost())
	if err != nil {
		log.Printf("endorsorQuery NewEndorsorClient error:%v\n", err)
		return nil, err
	}
	defer conn.Close()
	cli := pb.NewXendorserClient(conn)
	endorsorRes, err := cli.EndorserCall(ctx, req)
	if err != nil {
		log.Printf("endorsorQuery EndorserCall error:%v\n", err)
		return nil, err
	}
	res := &pb.CrossQueryResponse{}
	err = json.Unmarshal(endorsorRes.GetResponseData(), res)
	if err != nil {
		log.Printf("endorsorQuery Unmarshal error:%v\n", err)
		return nil, err
	}
	queryRes := &queryRes{
		queryRes: res,
		signs:    endorsorRes.GetEndorserSign(),
	}
	return queryRes, nil
}

// 验证CrossQuery背书信息
func IsCossQueryValid(
	crossQueryRequest *pb.CrossQueryRequest,
	queryMeta *pb.CrossQueryMeta,
	queryInfo *pb.CrossQueryInfo) bool {
	// check req from client and req from preexec
	if !isMsgEqual(crossQueryRequest, queryInfo.GetRequest()) {
		log.Println("isCossQueryValid isMsgEqual not equal")
		return false
	}

	// check endorsor info
	signs, ok := isEndorsorInfoValid(queryMeta, queryInfo.GetSigns())
	if !ok {
		log.Println("isEndorsorInfoValid not ok")
		return false
	}

	// check endorsor sign
	if !isEndorsorSignValid(signs, queryInfo) {
		log.Println("isEndorsorSignValid not ok")
		return false
	}
	return true
}

func isEndorsorInfoValid(queryMeta *pb.CrossQueryMeta, signs []*pb.SignatureInfo) ([]*pb.SignatureInfo, bool) {
	signMap := map[string]*pb.SignatureInfo{}
	for idx := range signs {
		signMap[signs[idx].GetPublicKey()] = signs[idx]
	}
	endorsorMap := map[string]*pb.CrossEndorsor{}
	endorsors := queryMeta.GetEndorsors()
	for idx := range endorsors {
		endorsorMap[endorsors[idx].GetPubKey()] = endorsors[idx]
	}
	signsValid := []*pb.SignatureInfo{}
	for k, v := range signMap {
		if endorsorMap[k] != nil {
			signsValid = append(signsValid, v)
		}
	}
	if len(signsValid) < int(queryMeta.GetChainMeta().GetMinEndorsorNum()) {
		log.Println("isEndorsorInfoValid failed")
		return nil, false
	}
	return signsValid, true
}

func isEndorsorSignValid(signsValid []*pb.SignatureInfo, queryInfo *pb.CrossQueryInfo) bool {
	reqData, err := json.Marshal(queryInfo.GetRequest())
	if err != nil {
		log.Printf("Marshal Request failed:%v\n", err)
		return false
	}
	resData, err := json.Marshal(queryInfo.GetResponse())
	if err != nil {
		log.Printf("Marshal Response failed:%v\n", err)
		return false
	}
	cryptoClient := crypto_client.GetCryptoClient()
	data := append(reqData[:], resData[:]...)
	digest := UsingSha256(data)
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

func isCrossQueryResponseEqual(a, b *pb.CrossQueryResponse) bool {
	if a.GetResponse().GetStatus() != b.GetResponse().GetStatus() {
		return false
	}
	if a.GetResponse().GetMessage() != b.GetResponse().GetMessage() {
		return false
	}
	if !bytes.Equal(a.GetResponse().GetBody(), b.GetResponse().GetBody()) {
		return false
	}
	return true
}

func isMsgEqual(reqHead, reqIncome proto.Message) bool {
	encodeHead, err := encodeMsg(reqHead)
	if err != nil {
		return false
	}
	encodeIncome, err := encodeMsg(reqIncome)
	if err != nil {
		return false
	}
	return bytes.Equal(encodeHead, encodeIncome)
}

func encodeMsg(req proto.Message) ([]byte, error) {
	var buf proto.Buffer
	buf.SetDeterministic(true)
	err := buf.EncodeMessage(req)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateInvokeIR(args map[string]string, methodName, contractName string) *pb.InvokeRequest {
	return &pb.InvokeRequest{
		ModuleName:   "wasm",
		MethodName:   methodName,
		ContractName: contractName,
		Args:         convertToXuperContractArgs(args),
	}
}

func convertToXuperContractArgs(args map[string]string) map[string][]byte {
	argmap := make(map[string][]byte)
	for k, v := range args {
		argmap[k] = []byte(v)
	}
	return argmap
}

// UsingSha256 get the hash result of data using SHA256
func UsingSha256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	out := h.Sum(nil)

	return out
}
