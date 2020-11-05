package cross_query

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	"github.com/xuperchain/xuper-sdk-go/pb"
)

const (
	endorserTimeOut = 6 * time.Second
)

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

// isQueryParamValid 验证 query meta 背书策略是否有效
func isQueryMetaValid(queryMeta *pb.CrossQueryMeta) bool {
	return len(queryMeta.GetEndorsors()) >= int(queryMeta.GetChainMeta().GetMinEndorsorNum())
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

// usingSha256 get the hash result of data using SHA256
func usingSha256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	out := h.Sum(nil)

	return out
}
