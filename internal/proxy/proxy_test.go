package proxy

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/milvus-io/milvus/internal/proto/proxypb"

	"github.com/golang/protobuf/proto"

	"github.com/milvus-io/milvus/internal/proto/schemapb"

	"github.com/milvus-io/milvus/internal/proto/milvuspb"

	"github.com/milvus-io/milvus/internal/proto/commonpb"

	"github.com/milvus-io/milvus/internal/proto/internalpb"

	grpcindexcoordclient "github.com/milvus-io/milvus/internal/distributed/indexcoord/client"

	grpcquerycoordclient "github.com/milvus-io/milvus/internal/distributed/querycoord/client"

	grpcdatacoordclient2 "github.com/milvus-io/milvus/internal/distributed/datacoord/client"

	"github.com/milvus-io/milvus/internal/util/funcutil"
	"github.com/milvus-io/milvus/internal/util/typeutil"

	rcc "github.com/milvus-io/milvus/internal/distributed/rootcoord/client"

	grpcindexnode "github.com/milvus-io/milvus/internal/distributed/indexnode"

	grpcindexcoord "github.com/milvus-io/milvus/internal/distributed/indexcoord"

	grpcdatacoordclient "github.com/milvus-io/milvus/internal/distributed/datacoord"
	grpcdatanode "github.com/milvus-io/milvus/internal/distributed/datanode"

	grpcquerynode "github.com/milvus-io/milvus/internal/distributed/querynode"

	grpcquerycoord "github.com/milvus-io/milvus/internal/distributed/querycoord"

	grpcrootcoord "github.com/milvus-io/milvus/internal/distributed/rootcoord"

	"github.com/milvus-io/milvus/internal/datacoord"
	"github.com/milvus-io/milvus/internal/datanode"
	"github.com/milvus-io/milvus/internal/indexcoord"
	"github.com/milvus-io/milvus/internal/indexnode"
	"github.com/milvus-io/milvus/internal/querynode"

	"github.com/milvus-io/milvus/internal/querycoord"
	"github.com/stretchr/testify/assert"

	"github.com/milvus-io/milvus/internal/msgstream"

	"github.com/milvus-io/milvus/internal/log"
	"github.com/milvus-io/milvus/internal/logutil"
	"github.com/milvus-io/milvus/internal/metrics"
	"github.com/milvus-io/milvus/internal/rootcoord"
)

const (
	attempts      = 1000000
	sleepDuration = time.Millisecond * 200
)

func newMsgFactory(localMsg bool) msgstream.Factory {
	if localMsg {
		return msgstream.NewRmsFactory()
	}
	return msgstream.NewPmsFactory()
}

func runRootCoord(ctx context.Context, localMsg bool) *grpcrootcoord.Server {
	var rc *grpcrootcoord.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		rootcoord.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&rootcoord.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		rc, err := grpcrootcoord.NewServer(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = rc.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterRootCoord()
	return rc
}

func runQueryCoord(ctx context.Context, localMsg bool) *grpcquerycoord.Server {
	var qs *grpcquerycoord.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		querycoord.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&querycoord.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		qs, err = grpcquerycoord.NewServer(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = qs.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterQueryCoord()
	return qs
}

func runQueryNode(ctx context.Context, localMsg bool, alias string) *grpcquerynode.Server {
	var qn *grpcquerynode.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		querynode.Params.InitAlias(alias)
		querynode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&querynode.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		qn, err = grpcquerynode.NewServer(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = qn.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterQueryNode()
	return qn
}

func runDataCoord(ctx context.Context, localMsg bool) *grpcdatacoordclient.Server {
	var ds *grpcdatacoordclient.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		datacoord.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&datacoord.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		ds, err = grpcdatacoordclient.NewServer(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = ds.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterDataCoord()
	return ds
}

func runDataNode(ctx context.Context, localMsg bool, alias string) *grpcdatanode.Server {
	var dn *grpcdatanode.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		datanode.Params.InitAlias(alias)
		datanode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&datanode.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		dn, err = grpcdatanode.NewServer(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = dn.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterDataNode()
	return dn
}

func runIndexCoord(ctx context.Context, localMsg bool) *grpcindexcoord.Server {
	var is *grpcindexcoord.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		indexcoord.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&indexcoord.Params.Log)
			defer log.Sync()
		}

		var err error
		is, err = grpcindexcoord.NewServer(ctx)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = is.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterIndexCoord()
	return is
}

func runIndexNode(ctx context.Context, localMsg bool, alias string) *grpcindexnode.Server {
	var in *grpcindexnode.Server
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		indexnode.Params.InitAlias(alias)
		indexnode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&indexnode.Params.Log)
			defer log.Sync()
		}

		var err error
		in, err = grpcindexnode.NewServer(ctx)
		if err != nil {
			panic(err)
		}
		wg.Done()
		err = in.Run()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	metrics.RegisterIndexNode()
	return in
}

func TestProxy(t *testing.T) {
	var err error

	err = os.Setenv("ROCKSMQ_PATH", "/tmp/milvus/rocksmq")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	localMsg := true
	factory := newMsgFactory(localMsg)
	alias := "TestProxy"

	rc := runRootCoord(ctx, localMsg)
	log.Info("running root coordinator ...")

	if rc != nil {
		defer func() {
			err := rc.Stop()
			assert.NoError(t, err)
			log.Info("stop root coordinator")
		}()
	}

	dc := runDataCoord(ctx, localMsg)
	log.Info("running data coordinator ...")

	if dc != nil {
		defer func() {
			err := dc.Stop()
			assert.NoError(t, err)
			log.Info("stop data coordinator")
		}()
	}

	dn := runDataNode(ctx, localMsg, alias)
	log.Info("running data node ...")

	if dn != nil {
		defer func() {
			err := dn.Stop()
			assert.NoError(t, err)
			log.Info("stop data node")
		}()
	}

	qc := runQueryCoord(ctx, localMsg)
	log.Info("running query coordinator ...")

	if qc != nil {
		defer func() {
			err := qc.Stop()
			assert.NoError(t, err)
			log.Info("stop query coordinator")
		}()
	}

	qn := runQueryNode(ctx, localMsg, alias)
	log.Info("running query node ...")

	if qn != nil {
		defer func() {
			err := qn.Stop()
			assert.NoError(t, err)
			log.Info("stop query node")
		}()
	}

	ic := runIndexCoord(ctx, localMsg)
	log.Info("running index coordinator ...")

	if ic != nil {
		defer func() {
			err := ic.Stop()
			assert.NoError(t, err)
			log.Info("stop index coordinator")
		}()
	}

	in := runIndexNode(ctx, localMsg, alias)
	log.Info("running index node ...")

	if in != nil {
		defer func() {
			err := in.Stop()
			assert.NoError(t, err)
			log.Info("stop index node")
		}()
	}

	time.Sleep(time.Second)

	proxy, err := NewProxy(ctx, factory)
	assert.NoError(t, err)
	assert.NotNil(t, proxy)
	Params.Init()
	log.Info("Initialize parameter table of proxy")

	// register proxy
	err = proxy.Register()
	assert.NoError(t, err)
	log.Info("Register proxy done")

	rootCoordClient, err := rcc.NewClient(ctx, Params.MetaRootPath, Params.EtcdEndpoints)
	assert.NoError(t, err)
	err = rootCoordClient.Init()
	assert.NoError(t, err)
	err = funcutil.WaitForComponentHealthy(ctx, rootCoordClient, typeutil.RootCoordRole, attempts, sleepDuration)
	assert.NoError(t, err)
	proxy.SetRootCoordClient(rootCoordClient)
	log.Info("Proxy set root coordinator client")

	dataCoordClient, err := grpcdatacoordclient2.NewClient(ctx, Params.MetaRootPath, Params.EtcdEndpoints)
	assert.NoError(t, err)
	err = dataCoordClient.Init()
	assert.NoError(t, err)
	err = funcutil.WaitForComponentHealthy(ctx, dataCoordClient, typeutil.DataCoordRole, attempts, sleepDuration)
	assert.NoError(t, err)
	proxy.SetDataCoordClient(dataCoordClient)
	log.Info("Proxy set data coordinator client")

	queryCoordClient, err := grpcquerycoordclient.NewClient(ctx, Params.MetaRootPath, Params.EtcdEndpoints)
	assert.NoError(t, err)
	err = queryCoordClient.Init()
	assert.NoError(t, err)
	err = funcutil.WaitForComponentHealthy(ctx, queryCoordClient, typeutil.QueryCoordRole, attempts, sleepDuration)
	assert.NoError(t, err)
	proxy.SetQueryCoordClient(queryCoordClient)
	log.Info("Proxy set query coordinator client")

	indexCoordClient, err := grpcindexcoordclient.NewClient(ctx, Params.MetaRootPath, Params.EtcdEndpoints)
	assert.NoError(t, err)
	err = indexCoordClient.Init()
	assert.NoError(t, err)
	err = funcutil.WaitForComponentHealthy(ctx, indexCoordClient, typeutil.IndexCoordRole, attempts, sleepDuration)
	assert.NoError(t, err)
	proxy.SetIndexCoordClient(indexCoordClient)
	log.Info("Proxy set index coordinator client")

	proxy.UpdateStateCode(internalpb.StateCode_Initializing)

	err = proxy.Init()
	assert.NoError(t, err)

	err = proxy.Start()
	assert.NoError(t, err)
	assert.Equal(t, internalpb.StateCode_Healthy, proxy.stateCode.Load().(internalpb.StateCode))
	defer func() {
		err := proxy.Stop()
		assert.NoError(t, err)
	}()

	t.Run("get component states", func(t *testing.T) {
		states, err := proxy.GetComponentStates(ctx)
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, states.Status.ErrorCode)
		assert.Equal(t, Params.ProxyID, states.State.NodeID)
		assert.Equal(t, typeutil.ProxyRole, states.State.Role)
		assert.Equal(t, proxy.stateCode.Load().(internalpb.StateCode), states.State.StateCode)
	})

	t.Run("get statistics channel", func(t *testing.T) {
		resp, err := proxy.GetStatisticsChannel(ctx)
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, resp.Status.ErrorCode)
		assert.Equal(t, "", resp.Value)
	})

	prefix := "test_proxy_"
	dbName := ""
	collectionName := prefix + funcutil.GenRandomStr()
	shardsNum := int32(2)
	int64Field := "int64"
	floatVecField := "fVec"
	dim := 128

	// an int64 field (pk) & a float vector field
	constructCollectionSchema := func() *schemapb.CollectionSchema {
		pk := &schemapb.FieldSchema{
			FieldID:      0,
			Name:         int64Field,
			IsPrimaryKey: true,
			Description:  "",
			DataType:     schemapb.DataType_Int64,
			TypeParams:   nil,
			IndexParams:  nil,
			AutoID:       true,
		}
		fVec := &schemapb.FieldSchema{
			FieldID:      0,
			Name:         floatVecField,
			IsPrimaryKey: false,
			Description:  "",
			DataType:     schemapb.DataType_FloatVector,
			TypeParams: []*commonpb.KeyValuePair{
				{
					Key:   "dim",
					Value: strconv.Itoa(dim),
				},
			},
			IndexParams: nil,
			AutoID:      false,
		}
		return &schemapb.CollectionSchema{
			Name:        collectionName,
			Description: "",
			AutoID:      false,
			Fields: []*schemapb.FieldSchema{
				pk,
				fVec,
			},
		}
	}

	constructCreateCollectionRequest := func() *milvuspb.CreateCollectionRequest {
		schema := constructCollectionSchema()
		bs, err := proto.Marshal(schema)
		assert.NoError(t, err)
		return &milvuspb.CreateCollectionRequest{
			Base:           nil,
			DbName:         dbName,
			CollectionName: collectionName,
			Schema:         bs,
			ShardsNum:      shardsNum,
		}
	}

	t.Run("create collection", func(t *testing.T) {
		req := constructCreateCollectionRequest()
		resp, err := proxy.CreateCollection(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, resp.ErrorCode)
	})

	t.Run("drop collection", func(t *testing.T) {
		collectionID, err := globalMetaCache.GetCollectionID(ctx, collectionName)
		assert.NoError(t, err)

		resp, err := proxy.DropCollection(ctx, &milvuspb.DropCollectionRequest{
			DbName:         dbName,
			CollectionName: collectionName,
		})
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, resp.ErrorCode)

		// invalidate meta cache
		resp, err = proxy.InvalidateCollectionMetaCache(ctx, &proxypb.InvalidateCollMetaCacheRequest{
			Base:           nil,
			DbName:         dbName,
			CollectionName: collectionName,
		})
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, resp.ErrorCode)

		// release dql stream
		resp, err = proxy.ReleaseDQLMessageStream(ctx, &proxypb.ReleaseDQLMessageStreamRequest{
			Base:         nil,
			DbID:         0,
			CollectionID: collectionID,
		})
		assert.NoError(t, err)
		assert.Equal(t, commonpb.ErrorCode_Success, resp.ErrorCode)
	})

	cancel()
}