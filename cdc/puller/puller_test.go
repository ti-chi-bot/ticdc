// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package puller

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/pingcap/errors"
	tidbkv "github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/store/mockstore"
	"github.com/pingcap/tiflow/cdc/kv"
	"github.com/pingcap/tiflow/cdc/model"
	"github.com/pingcap/tiflow/pkg/config"
	cerrors "github.com/pingcap/tiflow/pkg/errors"
	"github.com/pingcap/tiflow/pkg/pdutil"
	"github.com/pingcap/tiflow/pkg/regionspan"
	"github.com/pingcap/tiflow/pkg/retry"
	"github.com/pingcap/tiflow/pkg/security"
	"github.com/pingcap/tiflow/pkg/txnutil"
	"github.com/stretchr/testify/require"
	"github.com/tikv/client-go/v2/tikv"
	pd "github.com/tikv/pd/client"
)

type mockPdClientForPullerTest struct {
	pd.Client
	clusterID uint64
}

func (mc *mockPdClientForPullerTest) GetClusterID(ctx context.Context) uint64 {
	return mc.clusterID
}

type mockCDCKVClient struct {
	kv.CDCKVClient
	expectations chan model.RegionFeedEvent
}

type mockInjectedPuller struct {
	Puller
	cli *mockCDCKVClient
}

func newMockCDCKVClient(
	ctx context.Context,
	pd pd.Client,
	grpcPool kv.GrpcPool,
	regionCache *tikv.RegionCache,
	pdClock pdutil.Clock,
	cfg *config.ServerConfig,
	changefeed model.ChangeFeedID,
	tableID model.TableID,
	tableName string,
	filterloop bool,
) kv.CDCKVClient {
	return &mockCDCKVClient{
		expectations: make(chan model.RegionFeedEvent, 1024),
	}
}

func (mc *mockCDCKVClient) EventFeed(
	ctx context.Context,
	span regionspan.ComparableSpan,
	ts uint64,
	lockResolver txnutil.LockResolver,
	eventCh chan<- model.RegionFeedEvent,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-mc.expectations:
			if !ok {
				return nil
			}
			eventCh <- ev
		}
	}
}

func (mc *mockCDCKVClient) Close() error {
	close(mc.expectations)
	if len(mc.expectations) > 0 {
		buf := bytes.NewBufferString("mockCDCKVClient: not all expectations were satisfied! Still waiting\n")
		for e := range mc.expectations {
			_, _ = buf.WriteString(fmt.Sprintf("%s", e.GetValue()))
		}
		return errors.New(buf.String())
	}
	return nil
}

func (mc *mockCDCKVClient) Returns(ev model.RegionFeedEvent) {
	mc.expectations <- ev
}

func newPullerForTest(
	t *testing.T,
	spans []regionspan.Span,
	checkpointTs uint64,
) (*mockInjectedPuller, context.CancelFunc, *sync.WaitGroup, tidbkv.Storage) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	store, err := mockstore.NewMockStore()
	require.Nil(t, err)
	backupNewCDCKVClient := kv.NewCDCKVClient
	kv.NewCDCKVClient = newMockCDCKVClient
	defer func() {
		kv.NewCDCKVClient = backupNewCDCKVClient
	}()
	pdCli := &mockPdClientForPullerTest{clusterID: uint64(1)}
	grpcPool := kv.NewGrpcPoolImpl(ctx, &security.Credential{})
	defer grpcPool.Close()
	regionCache := tikv.NewRegionCache(pdCli)
	defer regionCache.Close()
	plr := New(
		ctx, pdCli, grpcPool, regionCache, store, pdutil.NewClock4Test(),
		checkpointTs, spans, config.GetDefaultServerConfig(),
		model.DefaultChangeFeedID("changefeed-id-test"), 0, "table-test", false)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := plr.Run(ctx)
		if err != nil {
			require.Equal(t, context.Canceled, errors.Cause(err))
		}
	}()
	require.Nil(t, err)
	mockPlr := &mockInjectedPuller{
		Puller: plr,
		cli:    plr.(*pullerImpl).kvCli.(*mockCDCKVClient),
	}
	return mockPlr, cancel, &wg, store
}

func TestPullerResolvedForward(t *testing.T) {
	spans := []regionspan.Span{
		{Start: []byte("t_a"), End: []byte("t_e")},
	}
	checkpointTs := uint64(996)
	plr, cancel, wg, store := newPullerForTest(t, spans, checkpointTs)

	plr.cli.Returns(model.RegionFeedEvent{
		Resolved: &model.ResolvedSpans{
			Spans: []model.RegionComparableSpan{{
				Span: regionspan.ToComparableSpan(regionspan.Span{Start: []byte("t_a"), End: []byte("t_c")}),
			}}, ResolvedTs: uint64(1001),
		},
	})
	plr.cli.Returns(model.RegionFeedEvent{
		Resolved: &model.ResolvedSpans{
			Spans: []model.RegionComparableSpan{{
				Span: regionspan.ToComparableSpan(regionspan.Span{Start: []byte("t_c"), End: []byte("t_d")}),
			}}, ResolvedTs: uint64(1002),
		},
	})
	plr.cli.Returns(model.RegionFeedEvent{
		Resolved: &model.ResolvedSpans{
			Spans: []model.RegionComparableSpan{{
				Span: regionspan.ToComparableSpan(regionspan.Span{Start: []byte("t_d"), End: []byte("t_e")}),
			}}, ResolvedTs: uint64(1000),
		},
	})
	ev := <-plr.Output()
	require.Equal(t, model.OpTypeResolved, ev.OpType)
	require.Equal(t, uint64(1000), ev.CRTs)
	err := retry.Do(context.Background(), func() error {
		ts := plr.GetResolvedTs()
		if ts != uint64(1000) {
			return errors.Errorf("resolved ts %d of puller does not forward to 1000", ts)
		}
		return nil
	}, retry.WithBackoffBaseDelay(10), retry.WithMaxTries(10), retry.WithIsRetryableErr(cerrors.IsRetryableError))

	require.Nil(t, err)

	store.Close()
	cancel()
	wg.Wait()
}

func TestPullerRawKV(t *testing.T) {
	spans := []regionspan.Span{
		{Start: []byte("c"), End: []byte("e")},
	}
	checkpointTs := uint64(996)
	plr, cancel, wg, store := newPullerForTest(t, spans, checkpointTs)

	plr.cli.Returns(model.RegionFeedEvent{
		Val: &model.RawKVEntry{
			OpType: model.OpTypePut,
			Key:    []byte("a"),
			Value:  []byte("test-value"),
			CRTs:   uint64(1002),
		},
	})
	plr.cli.Returns(model.RegionFeedEvent{
		Val: &model.RawKVEntry{
			OpType: model.OpTypePut,
			Key:    []byte("d"),
			Value:  []byte("test-value"),
			CRTs:   uint64(1003),
		},
	})
	var ev *model.RawKVEntry
	ev = <-plr.Output()
	require.Equal(t, model.OpTypePut, ev.OpType)
	require.Equal(t, []byte("a"), ev.Key)
	ev = <-plr.Output()
	require.Equal(t, model.OpTypePut, ev.OpType)
	require.Equal(t, []byte("d"), ev.Key)

	store.Close()
	cancel()
	wg.Wait()
}
<<<<<<< HEAD
=======

type fakeFrontier struct {
	resolvedTs uint64
}

func (f *fakeFrontier) Forward(regionID uint64, span tablepb.Span, ts uint64) {
}

func (f *fakeFrontier) Frontier() uint64 {
	return f.resolvedTs
}

func (f *fakeFrontier) String() string {
	return ""
}

func (f *fakeFrontier) SpanString(span tablepb.Span) string {
	return ""
}

func (f *fakeFrontier) Entries(fn func(key []byte, ts uint64)) {
}

func TestDetectResolvedTsStuck(t *testing.T) {
	plr := &pullerImpl{}
	plr.cfg = &config.ServerConfig{
		Debug: &config.DebugConfig{Puller: &config.PullerConfig{
			EnableResolvedTsStuckDetection: false,
			ResolvedTsStuckInterval:        0,
		}},
	}
	// detectResolvedTsStuck should return nil if the detection is disabled.
	require.Nil(t, plr.detectResolvedTsStuck())

	plr.cfg.Debug.Puller.EnableResolvedTsStuckDetection = true
	plr.cfg.Debug.Puller.ResolvedTsStuckInterval = config.TomlDuration(time.Duration(5) * time.Minute)
	plr.startResolvedTs = 10
	tsTrack := &fakeFrontier{resolvedTs: 10}
	plr.tsTracker = tsTrack
	plr.lastForwardTime = time.Now().Add(-10 * time.Minute)
	// detectResolvedTsStuck should return nil if the resolved ts is not forward
	require.Nil(t, plr.detectResolvedTsStuck())

	plr.lastForwardTime = time.Now()
	tsTrack.resolvedTs = 20
	// detectResolvedTsStuck should return nil if the resolved ts is forward but not stuck
	require.Nil(t, plr.detectResolvedTsStuck())
	plr.lastForwardTime = time.Now().Add(-5 * time.Minute)
	require.NotNil(t, plr.detectResolvedTsStuck())
}
>>>>>>> 7f8f280b7a (puller(ticdc): add more log for puller (#10339))
