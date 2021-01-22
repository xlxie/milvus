package datanode

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/clientv3"

	etcdkv "github.com/zilliztech/milvus-distributed/internal/kv/etcd"
	"github.com/zilliztech/milvus-distributed/internal/msgstream"
	"github.com/zilliztech/milvus-distributed/internal/proto/commonpb"
	"github.com/zilliztech/milvus-distributed/internal/proto/internalpb2"
	"github.com/zilliztech/milvus-distributed/internal/util/flowgraph"
)

func newMetaTable() *metaTable {
	etcdClient, _ := clientv3.New(clientv3.Config{Endpoints: []string{Params.EtcdAddress}})

	etcdKV := etcdkv.NewEtcdKV(etcdClient, Params.MetaRootPath)
	mt, _ := NewMetaTable(etcdKV)
	return mt
}

func TestFlowGraphDDNode_Operate(t *testing.T) {
	newMeta()
	const ctxTimeInMillisecond = 2000
	const closeWithDeadline = false
	var ctx context.Context

	if closeWithDeadline {
		var cancel context.CancelFunc
		d := time.Now().Add(ctxTimeInMillisecond * time.Millisecond)
		ctx, cancel = context.WithDeadline(context.Background(), d)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	inFlushCh := make(chan *flushMsg, 10)
	defer close(inFlushCh)

	testPath := "/test/datanode/root/meta"
	err := clearEtcd(testPath)
	require.NoError(t, err)
	Params.MetaRootPath = testPath

	Params.FlushDdBufferSize = 4
	replica := newReplica()
	ddNode := newDDNode(ctx, newMetaTable(), inFlushCh, replica)

	colID := UniqueID(0)
	colName := "col-test-0"
	// create collection
	createColReq := internalpb2.CreateCollectionRequest{
		Base: &commonpb.MsgBase{
			MsgType:   commonpb.MsgType_kCreateCollection,
			MsgID:     1,
			Timestamp: 1,
			SourceID:  1,
		},
		CollectionID: colID,
		Schema:       make([]byte, 0),
	}
	createColMsg := msgstream.CreateCollectionMsg{
		BaseMsg: msgstream.BaseMsg{
			BeginTimestamp: Timestamp(1),
			EndTimestamp:   Timestamp(1),
			HashValues:     []uint32{uint32(0)},
		},
		CreateCollectionRequest: createColReq,
	}

	// drop collection
	dropColReq := internalpb2.DropCollectionRequest{
		Base: &commonpb.MsgBase{
			MsgType:   commonpb.MsgType_kDropCollection,
			MsgID:     2,
			Timestamp: 2,
			SourceID:  2,
		},
		CollectionID:   colID,
		CollectionName: colName,
	}
	dropColMsg := msgstream.DropCollectionMsg{
		BaseMsg: msgstream.BaseMsg{
			BeginTimestamp: Timestamp(2),
			EndTimestamp:   Timestamp(2),
			HashValues:     []uint32{uint32(0)},
		},
		DropCollectionRequest: dropColReq,
	}

	partitionID := UniqueID(100)
	partitionTag := "partition-test-0"
	// create partition
	createPartitionReq := internalpb2.CreatePartitionRequest{
		Base: &commonpb.MsgBase{
			MsgType:   commonpb.MsgType_kCreatePartition,
			MsgID:     3,
			Timestamp: 3,
			SourceID:  3,
		},
		CollectionID:   colID,
		PartitionID:    partitionID,
		CollectionName: colName,
		PartitionName:  partitionTag,
	}
	createPartitionMsg := msgstream.CreatePartitionMsg{
		BaseMsg: msgstream.BaseMsg{
			BeginTimestamp: Timestamp(3),
			EndTimestamp:   Timestamp(3),
			HashValues:     []uint32{uint32(0)},
		},
		CreatePartitionRequest: createPartitionReq,
	}

	// drop partition
	dropPartitionReq := internalpb2.DropPartitionRequest{
		Base: &commonpb.MsgBase{
			MsgType:   commonpb.MsgType_kDropPartition,
			MsgID:     4,
			Timestamp: 4,
			SourceID:  4,
		},
		CollectionID:   colID,
		PartitionID:    partitionID,
		CollectionName: colName,
		PartitionName:  partitionTag,
	}
	dropPartitionMsg := msgstream.DropPartitionMsg{
		BaseMsg: msgstream.BaseMsg{
			BeginTimestamp: Timestamp(4),
			EndTimestamp:   Timestamp(4),
			HashValues:     []uint32{uint32(0)},
		},
		DropPartitionRequest: dropPartitionReq,
	}

	inFlushCh <- &flushMsg{
		msgID:        1,
		Timestamp:    6,
		segmentIDs:   []UniqueID{1},
		collectionID: UniqueID(1),
	}

	tsMessages := make([]msgstream.TsMsg, 0)
	tsMessages = append(tsMessages, msgstream.TsMsg(&createColMsg))
	tsMessages = append(tsMessages, msgstream.TsMsg(&dropColMsg))
	tsMessages = append(tsMessages, msgstream.TsMsg(&createPartitionMsg))
	tsMessages = append(tsMessages, msgstream.TsMsg(&dropPartitionMsg))
	msgStream := flowgraph.GenerateMsgStreamMsg(tsMessages, Timestamp(0), Timestamp(3))
	var inMsg Msg = msgStream
	ddNode.Operate([]*Msg{&inMsg})
}

func clearEtcd(rootPath string) error {
	etcdAddr := Params.EtcdAddress
	etcdClient, err := clientv3.New(clientv3.Config{Endpoints: []string{etcdAddr}})
	if err != nil {
		return err
	}
	etcdKV := etcdkv.NewEtcdKV(etcdClient, rootPath)

	err = etcdKV.RemoveWithPrefix("writer/segment")
	if err != nil {
		return err
	}
	_, _, err = etcdKV.LoadWithPrefix("writer/segment")
	if err != nil {
		return err
	}
	log.Println("Clear ETCD with prefix writer/segment ")

	err = etcdKV.RemoveWithPrefix("writer/ddl")
	if err != nil {
		return err
	}
	_, _, err = etcdKV.LoadWithPrefix("writer/ddl")
	if err != nil {
		return err
	}
	log.Println("Clear ETCD with prefix writer/ddl")
	return nil

}
