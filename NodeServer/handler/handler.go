package handler

import (
	. "Puzzle/Logger"
	"Puzzle/NodeServer/utils"
	"Puzzle/Storage"
	pb "Puzzle/idl"
	"context"
	"encoding/binary"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"sync"
	"sync/atomic"
)

type Handler struct {
	pb.UnimplementedCacheServiceServer
	*Storage.StorageService
}

func (s *Handler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	log.Println("Received: " + req.Payload)
	return &pb.PingResponse{Message:"Hello client"}, nil
}

func (s *Handler) SetValues(ctx context.Context, req *pb.SetValuesRequest) (*pb.SetValuesResponse, error) {
	if len(req.Cells) == 0 {
		return &pb.SetValuesResponse{Code:0, Message:"ok"}, nil
	}
	if !utils.CfContains(string(req.Cells[0].ColumnFamily)) {
		//meta data should not be cached
		return &pb.SetValuesResponse{Code:0, Message:"ok"}, nil
	}
	mSetMap := map[string]map[string]interface{}{}
	Logger.Debugf("Set cells for:" + string(req.Cells[0].Row))
	for _, cell := range req.Cells {
		// todo: check whether these cells contains stale data(compare timestamp)
		// encode here
		timestampBytes := make([]byte, 8)
		typeBytes := make([]byte, 4)
		binary.LittleEndian.PutUint64(timestampBytes, uint64(cell.Timestamp))
		binary.LittleEndian.PutUint32(typeBytes, uint32(cell.Type))
		merged := append(timestampBytes, typeBytes...)
		key := string(append(req.Cells[0].Row, req.Cells[0].ColumnFamily...))
		if _, ex := mSetMap[key]; !ex {
			mSetMap[key] = make(map[string]interface{})
		}
		mSetMap[key][string(cell.Column)] = string(append(cell.Value, merged...))
	}
	var wg sync.WaitGroup
	var eVal atomic.Value
	for k, v := range mSetMap {
		go func() {
			wg.Add(1)
			defer wg.Done()
			err := s.StorageService.HMSet(k, v)
			if err != nil {
				Logger.Errorf("SetValues error: %v", err)
				eVal.Store(err)
			}
		}()
	}
	wg.Wait()
	if eVal.Load() != nil {
		return nil, (eVal.Load()).(error)
	}
	return &pb.SetValuesResponse{Code:0, Message:"ok"}, nil
}

type compRes struct {
	val map[string]string
	err error
	cf string
}

func (s *Handler) GetRow(ctx context.Context, req *pb.GetRowRequest) (*pb.GetRowResponse, error) {
	Logger.Debugf("Get row for key:" + string(req.Key))
	cfMap := utils.GetShortCfMapCopy()
	result := make([]*pb.HCell, 0)
	resChan := make(chan compRes, len(cfMap))
	for _, v := range cfMap{
		go func() {
			res, err := s.HGetAll(string(req.Key) + v)
			resChan <- compRes{res, err,v }
		}()

	}

	for i:=0; i < len(cfMap); i++ {
		cRes := <-resChan
		res := cRes.val
		err := cRes.err
		if err != nil && err.Error() == "deleting"{
			return &pb.GetRowResponse{Code:1, Result:nil, Message:"no records in cache"}, nil
		}
		if err != nil {
			Logger.Errorf("GetRow error: %v", err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		if len(res) != 0 {
			for col, val := range res {
				val := []byte(val)
				timestampBytes := val[len(val) - 12: len(val) - 4]
				typeBytes := val[len(val)-4:]
				typeVal := int32(binary.LittleEndian.Uint32(typeBytes))
				if typeVal != 4{
					// deleted, skip it
					continue
				}
				result = append(result, &pb.HCell{
					Row:req.Key,
					ColumnFamily: []byte(cRes.cf),
					Column:[]byte(col),
					Value:val[0: len(val) - 12],
					Timestamp: int64(binary.LittleEndian.Uint64(timestampBytes)),
					Type: typeVal,
				})
			}
		}
	}
	if len(result) == 0 {
		return &pb.GetRowResponse{Code:1, Result:nil, Message:"no records in cache"}, nil
	}
	return &pb.GetRowResponse{Code:0, Result:result, Message:"ok"}, nil
}