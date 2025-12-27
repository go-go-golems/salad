package saleae

import (
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
)

type CaptureStatus int

type CaptureOrigin int

type CaptureModeKind int

type CloseCaptureMode int

type WaitCapturePolicy int

const (
	CaptureStatusRunning CaptureStatus = iota
	CaptureStatusStopped
	CaptureStatusCompleted
	CaptureStatusClosed
)

const (
	CaptureOriginLoaded CaptureOrigin = iota
	CaptureOriginStarted
)

const (
	CaptureModeTimed CaptureModeKind = iota
	CaptureModeManual
	CaptureModeTrigger
)

const (
	CloseCaptureDelete CloseCaptureMode = iota
	CloseCaptureMarkClosed
)

const (
	WaitCaptureImmediate WaitCapturePolicy = iota
	WaitCaptureErrorIfRunning
	WaitCaptureBlockUntilDone
)

type CaptureMode struct {
	Kind     CaptureModeKind
	Duration time.Duration
}

type CaptureState struct {
	ID        uint64
	Status    CaptureStatus
	Origin    CaptureOrigin
	StartedAt time.Time
	Mode      CaptureMode
}

type State struct {
	AppInfo       *pb.AppInfo
	Devices       []*pb.Device
	Captures      map[uint64]*CaptureState
	NextCaptureID uint64
}
