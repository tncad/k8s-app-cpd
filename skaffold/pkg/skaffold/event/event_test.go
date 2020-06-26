/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package event

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 10000; step++ {
		ev := &eventHandler{}

		ev.logEvent(proto.LogEntry{Entry: "OLD"})
		go func() {
			ev.logEvent(proto.LogEntry{Entry: "FRESH"})
			ev.logEvent(proto.LogEntry{Entry: "POISON PILL"})
		}()

		var received int32
		ev.forEachEvent(func(e *proto.LogEntry) error {
			if e.Entry == "POISON PILL" {
				return errors.New("Done")
			}

			atomic.AddInt32(&received, 1)
			return nil
		})

		if atomic.LoadInt32(&received) != 2 {
			t.Fatalf("Expected %d events, Got %d (Step: %d)", 2, received, step)
		}
	}
}

func TestGetState(t *testing.T) {
	ev := &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	ev.stateLock.Lock()
	ev.state.BuildState.Artifacts["img"] = Complete
	ev.stateLock.Unlock()

	state := ev.getState()

	testutil.CheckDeepEqual(t, Complete, state.BuildState.Artifacts["img"])
}

func TestDeployInProgress(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployInProgress()
	wait(t, func() bool { return handler.getState().DeployState.Status == InProgress })
}

func TestDeployFailed(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployFailed(errors.New("BUG"))
	wait(t, func() bool { return handler.getState().DeployState.Status == Failed })
}

func TestDeployComplete(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployComplete()
	wait(t, func() bool { return handler.getState().DeployState.Status == Complete })
}

func TestBuildInProgress(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildInProgress("img")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == InProgress })
}

func TestBuildFailed(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildFailed("img", errors.New("BUG"))
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == Failed })
}

func TestBuildComplete(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildComplete("img")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == Complete })
}

func TestPortForwarded(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().ForwardedPorts[8080] == nil })
	PortForwarded(8080, 8888, "pod", "container", "ns", "portname", "resourceType", "resourceName", "127.0.0.1")
	wait(t, func() bool { return handler.getState().ForwardedPorts[8080] != nil })
}

func TestStatusCheckEventStarted(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	StatusCheckEventStarted()
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == Started })
}

func TestStatusCheckEventInProgress(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	StatusCheckEventInProgress("[2/5 deployment(s) are still pending]")
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == InProgress })
}

func TestStatusCheckEventSucceeded(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	statusCheckEventSucceeded()
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == Succeeded })
}

func TestStatusCheckEventFailed(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	statusCheckEventFailed(errors.New("one or more deployments failed"))
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == Failed })
}

func TestResourceStatusCheckEventUpdated(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	ResourceStatusCheckEventUpdated("ns:pod/foo", "img pull error")
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == InProgress })
}

func TestResourceStatusCheckEventSucceeded(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventSucceeded("ns:pod/foo")
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Succeeded })
}

func TestResourceStatusCheckEventFailed(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventFailed("ns:pod/foo", errors.New("one or more deployments failed"))
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Failed })
}

func TestFileSyncInProgress(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncInProgress(5, "image")
	wait(t, func() bool { return handler.getState().FileSyncState.Status == InProgress })
}

func TestFileSyncFailed(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncFailed(5, "image", errors.New("BUG"))
	wait(t, func() bool { return handler.getState().FileSyncState.Status == Failed })
}

func TestFileSyncSucceeded(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncSucceeded(5, "image")
	wait(t, func() bool { return handler.getState().FileSyncState.Status == Succeeded })
}

func TestDebuggingContainer(t *testing.T) {
	defer func() { handler = &eventHandler{} }()

	handler = &eventHandler{
		state: emptyState(latest.Pipeline{}, "test", true, true, true),
	}

	found := func() bool {
		for _, dc := range handler.getState().DebuggingContainers {
			if dc.Namespace == "ns" && dc.PodName == "pod" && dc.ContainerName == "container" {
				return true
			}
		}
		return false
	}
	notFound := func() bool { return !found() }
	wait(t, notFound)
	DebuggingContainerStarted("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, found)
	DebuggingContainerTerminated("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, notFound)
}

func wait(t *testing.T, condition func() bool) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}

		case <-timeout.C:
			t.Fatal("Timed out waiting")
		}
	}
}

func TestResetStateOnBuild(t *testing.T) {
	defer func() { handler = &eventHandler{} }()
	handler = &eventHandler{
		state: proto.State{
			BuildState: &proto.BuildState{
				Artifacts: map[string]string{
					"image1": Complete,
				},
			},
			DeployState: &proto.DeployState{Status: Complete},
			ForwardedPorts: map[int32]*proto.PortEvent{
				2001: {
					LocalPort:  2000,
					RemotePort: 2001,
					PodName:    "test/pod",
				},
			},
			StatusCheckState: &proto.StatusCheckState{Status: Complete},
			FileSyncState:    &proto.FileSyncState{Status: Succeeded},
		},
	}
	ResetStateOnBuild()
	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": NotStarted,
			},
		},
		DeployState:      &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted, Resources: map[string]string{}},
		FileSyncState:    &proto.FileSyncState{Status: NotStarted},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}

func TestResetStateOnDeploy(t *testing.T) {
	defer func() { handler = &eventHandler{} }()
	handler = &eventHandler{
		state: proto.State{
			BuildState: &proto.BuildState{
				Artifacts: map[string]string{
					"image1": Complete,
				},
			},
			DeployState: &proto.DeployState{Status: Complete},
			ForwardedPorts: map[int32]*proto.PortEvent{
				2001: {
					LocalPort:  2000,
					RemotePort: 2001,
					PodName:    "test/pod",
				},
			},
			StatusCheckState: &proto.StatusCheckState{Status: Complete},
		},
	}
	ResetStateOnDeploy()
	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted,
			Resources: map[string]string{},
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}

func TestEmptyStateCheckState(t *testing.T) {
	actual := emptyStatusCheckState()
	expected := &proto.StatusCheckState{Status: NotStarted,
		Resources: map[string]string{},
	}
	testutil.CheckDeepEqual(t, expected, actual, cmpopts.EquateEmpty())
}

func TestUpdateStateAutoTriggers(t *testing.T) {
	defer func() { handler = &eventHandler{} }()
	handler = &eventHandler{
		state: proto.State{
			BuildState: &proto.BuildState{
				Artifacts: map[string]string{
					"image1": Complete,
				},
				AutoTrigger: false,
			},
			DeployState: &proto.DeployState{Status: Complete, AutoTrigger: false},
			ForwardedPorts: map[int32]*proto.PortEvent{
				2001: {
					LocalPort:  2000,
					RemotePort: 2001,
					PodName:    "test/pod",
				},
			},
			StatusCheckState: &proto.StatusCheckState{Status: Complete},
			FileSyncState: &proto.FileSyncState{
				Status:      "Complete",
				AutoTrigger: false,
			},
		},
	}
	UpdateStateAutoBuildTrigger(true)
	UpdateStateAutoDeployTrigger(true)
	UpdateStateAutoSyncTrigger(true)

	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: true,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: true},
		ForwardedPorts: map[int32]*proto.PortEvent{
			2001: {
				LocalPort:  2000,
				RemotePort: 2001,
				PodName:    "test/pod",
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState: &proto.FileSyncState{
			Status:      "Complete",
			AutoTrigger: true,
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}
