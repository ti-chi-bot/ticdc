// Copyright 2021 PingCAP, Inc.
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

package owner

import (
	"context"
	"fmt"
	"io"
<<<<<<< HEAD
	"math"
=======
	"net/url"
	"os"
>>>>>>> 1f6ae1fd8c (cdc: add delay for recreating changefeed (#7730))
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/failpoint"
	"github.com/pingcap/log"
	"github.com/pingcap/tiflow/cdc/model"
	cdcContext "github.com/pingcap/tiflow/pkg/context"
	cerror "github.com/pingcap/tiflow/pkg/errors"
	"github.com/pingcap/tiflow/pkg/orchestrator"
	"github.com/pingcap/tiflow/pkg/txnutil/gc"
	"github.com/pingcap/tiflow/pkg/version"
	pd "github.com/tikv/pd/client"
	"go.uber.org/zap"
)

type ownerJobType int

// All OwnerJob types
const (
	ownerJobTypeRebalance ownerJobType = iota
	ownerJobTypeManualSchedule
	ownerJobTypeAdminJob
	ownerJobTypeDebugInfo
)

<<<<<<< HEAD
=======
// versionInconsistentLogRate represents the rate of log output when there are
// captures with versions different from that of the owner
const versionInconsistentLogRate = 1

// Remove following variables once we fix https://github.com/pingcap/tiflow/issues/7657.
var (
	recreateChangefeedDelayLimit = 30 * time.Second
	hasCIEnv                     = func() bool {
		// Most CI platform has the "CI" environment variable.
		_, ok := os.LookupEnv("CI")
		return ok
	}()
)

// Export field names for pretty printing.
>>>>>>> 1f6ae1fd8c (cdc: add delay for recreating changefeed (#7730))
type ownerJob struct {
	tp           ownerJobType
	changefeedID model.ChangeFeedID

	// for ManualSchedule only
	targetCaptureID model.CaptureID
	// for ManualSchedule only
	tableID model.TableID

	// for Admin Job only
	adminJob *model.AdminJob

	// for debug info only
	debugInfoWriter io.Writer

	done chan struct{}
}

// Owner manages many changefeeds
// All public functions are THREAD-SAFE, except for Tick, Tick is only used for etcd worker
type Owner struct {
	changefeeds map[model.ChangeFeedID]*changefeed

	gcManager gc.Manager

<<<<<<< HEAD
	ownerJobQueueMu sync.Mutex
	ownerJobQueue   []*ownerJob
=======
// Owner managers TiCDC cluster.
//
// The interface is thread-safe, except for Tick, it's only used by etcd worker.
type Owner interface {
	orchestrator.Reactor
	EnqueueJob(adminJob model.AdminJob, done chan<- error)
	RebalanceTables(cfID model.ChangeFeedID, done chan<- error)
	ScheduleTable(
		cfID model.ChangeFeedID, toCapture model.CaptureID,
		tableID model.TableID, done chan<- error,
	)
	DrainCapture(query *scheduler.Query, done chan<- error)
	WriteDebugInfo(w io.Writer, done chan<- error)
	Query(query *Query, done chan<- error)
	ValidateChangefeed(info *model.ChangeFeedInfo) error
	AsyncStop()
}
>>>>>>> 1f6ae1fd8c (cdc: add delay for recreating changefeed (#7730))

	lastTickTime time.Time

	// bootstrapped specifies whether the owner has been initialized.
	// This will only be done when the owner starts the first Tick.
	// NOTICE: Do not use it in a method other than tick unexpectedly, as it is not a thread-safe value.
	bootstrapped bool
<<<<<<< HEAD

	closed int32

	newChangefeed func(id model.ChangeFeedID, gcManager gc.Manager) *changefeed
}

// NewOwner creates a new Owner
func NewOwner(pdClient pd.Client) *Owner {
	return &Owner{
		changefeeds:   make(map[model.ChangeFeedID]*changefeed),
		gcManager:     gc.NewManager(pdClient),
		lastTickTime:  time.Now(),
		newChangefeed: newChangefeed,
=======
	// changefeedTicked specifies whether changefeeds have been ticked.
	// NOTICE: Do not use it in a method other than tick unexpectedly,
	//         as it is not a thread-safe value.
	changefeedTicked bool

	newChangefeed func(
		id model.ChangeFeedID,
		state *orchestrator.ChangefeedReactorState,
		up *upstream.Upstream,
	) *changefeed

	// removedChangefeed is a workload of https://github.com/pingcap/tiflow/issues/7657
	// by delaying recreate changefeed with the same ID.
	// TODO: remove these fields after the issue is resolved.
	removedChangefeed map[model.ChangeFeedID]time.Time
	removedSinkURI    map[url.URL]time.Time
}

// NewOwner creates a new Owner
func NewOwner(upstreamManager *upstream.Manager) Owner {
	return &ownerImpl{
		upstreamManager:   upstreamManager,
		changefeeds:       make(map[model.ChangeFeedID]*changefeed),
		lastTickTime:      time.Now(),
		newChangefeed:     newChangefeed,
		logLimiter:        rate.NewLimiter(versionInconsistentLogRate, versionInconsistentLogRate),
		removedChangefeed: make(map[model.ChangeFeedID]time.Time),
		removedSinkURI:    make(map[url.URL]time.Time),
>>>>>>> 1f6ae1fd8c (cdc: add delay for recreating changefeed (#7730))
	}
}

// NewOwner4Test creates a new Owner for test
func NewOwner4Test(
	newDDLPuller func(ctx cdcContext.Context, startTs uint64) (DDLPuller, error),
	newSink func() DDLSink,
	pdClient pd.Client,
) *Owner {
	o := NewOwner(pdClient)
	// Most tests do not need to test bootstrap.
	o.bootstrapped = true
	o.newChangefeed = func(id model.ChangeFeedID, gcManager gc.Manager) *changefeed {
		return newChangefeed4Test(id, gcManager, newDDLPuller, newSink)
	}
	return o
}

// Tick implements the Reactor interface
func (o *Owner) Tick(stdCtx context.Context, rawState orchestrator.ReactorState) (nextState orchestrator.ReactorState, err error) {
	failpoint.Inject("owner-run-with-error", func() {
		failpoint.Return(nil, errors.New("owner run with injected error"))
	})
	failpoint.Inject("sleep-in-owner-tick", nil)
	state := rawState.(*model.GlobalReactorState)
	// At the first Tick, we need to do a bootstrap operation.
	// Fix incompatible or incorrect meta information.
	if !o.bootstrapped {
		o.Bootstrap(state)
		o.bootstrapped = true
		return state, nil
	}

	o.updateMetrics(state)
	if !o.clusterVersionConsistent(state.Captures) {
		// sleep one second to avoid printing too much log
		time.Sleep(1 * time.Second)
		return state, nil
	}

	// Owner should update GC safepoint before initializing changefeed, so
	// changefeed can remove its "ticdc-creating" service GC safepoint during
	// initializing.
	//
	// See more gc doc.
	if err = o.updateGCSafepoint(stdCtx, state); err != nil {
		return nil, errors.Trace(err)
	}

	o.handleJobs()
	ctx := stdCtx.(cdcContext.Context)
	for changefeedID, changefeedState := range state.Changefeeds {
		if changefeedState.Info == nil {
			o.cleanUpChangefeed(changefeedState)
			continue
		}
		ctx = cdcContext.WithChangefeedVars(ctx, &cdcContext.ChangefeedVars{
			ID:   changefeedID,
			Info: changefeedState.Info,
		})
		cfReactor, exist := o.changefeeds[changefeedID]
		if !exist {
			cfReactor = o.newChangefeed(changefeedID, o.gcManager)
			o.changefeeds[changefeedID] = cfReactor
		}
		cfReactor.Tick(ctx, changefeedState, state.Captures)
	}
	if len(o.changefeeds) != len(state.Changefeeds) {
		for changefeedID, cfReactor := range o.changefeeds {
			if _, exist := state.Changefeeds[changefeedID]; exist {
				continue
			}
			cfReactor.Close()
			delete(o.changefeeds, changefeedID)
		}
	}
	if atomic.LoadInt32(&o.closed) != 0 {
		for _, cfReactor := range o.changefeeds {
			cfReactor.Close()
		}
		return state, cerror.ErrReactorFinished.GenWithStackByArgs()
	}
	return state, nil
}

// EnqueueJob enqueues a admin job into a internal queue, and the Owner will handle the job in the next tick
func (o *Owner) EnqueueJob(adminJob model.AdminJob) {
	o.pushOwnerJob(&ownerJob{
		tp:           ownerJobTypeAdminJob,
		adminJob:     &adminJob,
		changefeedID: adminJob.CfID,
		done:         make(chan struct{}),
	})
}

// TriggerRebalance triggers a rebalance for the specified changefeed
func (o *Owner) TriggerRebalance(cfID model.ChangeFeedID) {
	o.pushOwnerJob(&ownerJob{
		tp:           ownerJobTypeRebalance,
		changefeedID: cfID,
		done:         make(chan struct{}),
	})
}

// ManualSchedule moves a table from a capture to another capture
func (o *Owner) ManualSchedule(cfID model.ChangeFeedID, toCapture model.CaptureID, tableID model.TableID) {
	o.pushOwnerJob(&ownerJob{
		tp:              ownerJobTypeManualSchedule,
		changefeedID:    cfID,
		targetCaptureID: toCapture,
		tableID:         tableID,
		done:            make(chan struct{}),
	})
}

// WriteDebugInfo writes debug info into the specified http writer
func (o *Owner) WriteDebugInfo(w io.Writer) {
	timeout := time.Second * 3
	done := make(chan struct{})
	o.pushOwnerJob(&ownerJob{
		tp:              ownerJobTypeDebugInfo,
		debugInfoWriter: w,
		done:            done,
	})
	// wait the debug info printed
	select {
	case <-done:
	case <-time.After(timeout):
		fmt.Fprintf(w, "failed to print debug info for owner\n")
	}
}

func (o *ownerImpl) ValidateChangefeed(info *model.ChangeFeedInfo) error {
	o.ownerJobQueue.Lock()
	defer o.ownerJobQueue.Unlock()
	if hasCIEnv {
		// Disable the check on CI platform, because many tests repeatedly
		// create changefeed with same name and same sinkURI.
		return nil
	}

	t, ok := o.removedChangefeed[model.ChangeFeedID{ID: info.ID, Namespace: info.Namespace}]
	if ok {
		remain := recreateChangefeedDelayLimit - time.Since(t)
		if remain >= 0 {
			return cerror.ErrInternalServerError.GenWithStackByArgs(fmt.Sprintf(
				"changefeed with same ID was just removed, please wait %s", remain))
		}
	}

	sinkURI, err := url.Parse(info.SinkURI)
	if err != nil {
		return cerror.ErrInternalServerError.GenWithStackByArgs(
			fmt.Sprintf("invalid sink URI %s", err))
	}
	t, ok = o.removedSinkURI[url.URL{
		Scheme: sinkURI.Scheme,
		Host:   sinkURI.Host,
	}]
	if ok {
		remain := recreateChangefeedDelayLimit - time.Since(t)
		if remain >= 0 {
			return cerror.ErrInternalServerError.GenWithStackByArgs(fmt.Sprintf(
				"changefeed with same sink URI was just removed, please wait %s", remain))
		}
	}
	return nil
}

// AsyncStop stops the owner asynchronously
func (o *Owner) AsyncStop() {
	atomic.StoreInt32(&o.closed, 1)
}

func (o *Owner) cleanUpChangefeed(state *model.ChangefeedReactorState) {
	state.PatchInfo(func(info *model.ChangeFeedInfo) (*model.ChangeFeedInfo, bool, error) {
		return nil, info != nil, nil
	})
	state.PatchStatus(func(status *model.ChangeFeedStatus) (*model.ChangeFeedStatus, bool, error) {
		return nil, status != nil, nil
	})
	for captureID := range state.TaskStatuses {
		state.PatchTaskStatus(captureID, func(status *model.TaskStatus) (*model.TaskStatus, bool, error) {
			return nil, status != nil, nil
		})
	}
	for captureID := range state.TaskPositions {
		state.PatchTaskPosition(captureID, func(position *model.TaskPosition) (*model.TaskPosition, bool, error) {
			return nil, position != nil, nil
		})
	}
	for captureID := range state.Workloads {
		state.PatchTaskWorkload(captureID, func(workload model.TaskWorkload) (model.TaskWorkload, bool, error) {
			return nil, workload != nil, nil
		})
	}
}

// Bootstrap checks if the state contains incompatible or incorrect information and tries to fix it.
func (o *Owner) Bootstrap(state *model.GlobalReactorState) {
	log.Info("Start bootstrapping", zap.Any("state", state))
	fixChangefeedInfos(state)
}

// fixChangefeedInfos attempts to fix incompatible or incorrect meta information in changefeed state.
func fixChangefeedInfos(state *model.GlobalReactorState) {
	for _, changefeedState := range state.Changefeeds {
		if changefeedState != nil {
			changefeedState.PatchInfo(func(info *model.ChangeFeedInfo) (*model.ChangeFeedInfo, bool, error) {
				if info == nil {
					return nil, false, nil
				}
				info.FixIncompatible()
				return info, true, nil
			})
		}
	}
}

func (o *Owner) updateMetrics(state *model.GlobalReactorState) {
	// Keep the value of prometheus expression `rate(counter)` = 1
	// Please also change alert rule in ticdc.rules.yml when change the expression value.
	now := time.Now()
	ownershipCounter.Add(float64(now.Sub(o.lastTickTime)) / float64(time.Second))
	o.lastTickTime = now

	ownerMaintainTableNumGauge.Reset()
	changefeedStatusGauge.Reset()
	for changefeedID, changefeedState := range state.Changefeeds {
		for captureID, captureInfo := range state.Captures {
			taskStatus, exist := changefeedState.TaskStatuses[captureID]
			if !exist {
				continue
			}
			ownerMaintainTableNumGauge.WithLabelValues(changefeedID, captureInfo.AdvertiseAddr, maintainTableTypeTotal).Set(float64(len(taskStatus.Tables)))
			ownerMaintainTableNumGauge.WithLabelValues(changefeedID, captureInfo.AdvertiseAddr, maintainTableTypeWip).Set(float64(len(taskStatus.Operation)))
			if changefeedState.Info != nil {
				changefeedStatusGauge.WithLabelValues(changefeedID).Set(float64(changefeedState.Info.State.ToInt()))
			}
		}
	}
}

func (o *Owner) clusterVersionConsistent(captures map[model.CaptureID]*model.CaptureInfo) bool {
	myVersion := version.ReleaseVersion
	for _, capture := range captures {
		if myVersion != capture.Version {
			log.Warn("the capture version is different with the owner", zap.Reflect("capture", capture), zap.String("my-version", myVersion))
			return false
		}
	}
	return true
}

func (o *Owner) handleJobs() {
	jobs := o.takeOwnerJobs()
	for _, job := range jobs {
		changefeedID := job.changefeedID
		cfReactor, exist := o.changefeeds[changefeedID]
		if !exist {
			log.Warn("changefeed not found when handle a job", zap.Reflect("job", job))
			continue
		}
		switch job.tp {
		case ownerJobTypeAdminJob:
<<<<<<< HEAD
			cfReactor.feedStateManager.PushAdminJob(job.adminJob)
		case ownerJobTypeManualSchedule:
			cfReactor.scheduler.MoveTable(job.tableID, job.targetCaptureID)
=======
			if job.AdminJob.Type == model.AdminRemove {
				now := time.Now()
				o.removedChangefeed[changefeedID] = now
				uri, err := url.Parse(cfReactor.state.Info.SinkURI)
				if err == nil {
					o.removedSinkURI[url.URL{
						Scheme: uri.Scheme,
						Host:   uri.Host,
					}] = now
				}
			}
			cfReactor.feedStateManager.PushAdminJob(job.AdminJob)
		case ownerJobTypeScheduleTable:
			// Scheduler is created lazily, it is nil before initialization.
			if cfReactor.scheduler != nil {
				cfReactor.scheduler.MoveTable(job.TableID, job.TargetCaptureID)
			}
		case ownerJobTypeDrainCapture:
			o.handleDrainCaptures(ctx, job.scheduleQuery, job.done)
			continue // continue here to prevent close the done channel twice
>>>>>>> 1f6ae1fd8c (cdc: add delay for recreating changefeed (#7730))
		case ownerJobTypeRebalance:
			cfReactor.scheduler.Rebalance()
		case ownerJobTypeDebugInfo:
			// TODO: implement this function
		}
		close(job.done)
	}

	// Try GC removed changefeed id/sink URI after delay limit passed.
	for id, t := range o.removedChangefeed {
		if time.Since(t) >= recreateChangefeedDelayLimit {
			delete(o.removedChangefeed, id)
		}
	}
	for s, t := range o.removedSinkURI {
		if time.Since(t) >= recreateChangefeedDelayLimit {
			delete(o.removedSinkURI, s)
		}
	}
}

func (o *Owner) takeOwnerJobs() []*ownerJob {
	o.ownerJobQueueMu.Lock()
	defer o.ownerJobQueueMu.Unlock()

	jobs := o.ownerJobQueue
	o.ownerJobQueue = nil
	return jobs
}

func (o *Owner) pushOwnerJob(job *ownerJob) {
	o.ownerJobQueueMu.Lock()
	defer o.ownerJobQueueMu.Unlock()
	o.ownerJobQueue = append(o.ownerJobQueue, job)
}

func (o *Owner) updateGCSafepoint(
	ctx context.Context, state *model.GlobalReactorState,
) error {
	forceUpdate := false
	minCheckpointTs := uint64(math.MaxUint64)
	for changefeedID, changefeefState := range state.Changefeeds {
		if changefeefState.Info == nil {
			continue
		}
		switch changefeefState.Info.State {
		case model.StateNormal, model.StateStopped, model.StateError:
		default:
			continue
		}
		checkpointTs := changefeefState.Info.GetCheckpointTs(changefeefState.Status)
		if minCheckpointTs > checkpointTs {
			minCheckpointTs = checkpointTs
		}
		// Force update when adding a new changefeed.
		_, exist := o.changefeeds[changefeedID]
		if !exist {
			forceUpdate = true
		}
	}
	// When the changefeed starts up, CDC will do a snapshot read at
	// (checkpointTs - 1) from TiKV, so (checkpointTs - 1) should be an upper
	// bound for the GC safepoint.
	gcSafepointUpperBound := minCheckpointTs - 1
	err := o.gcManager.TryUpdateGCSafePoint(ctx, gcSafepointUpperBound, forceUpdate)
	return errors.Trace(err)
}
