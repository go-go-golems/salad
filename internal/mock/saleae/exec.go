package saleae

import (
	"context"
	"sync"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
)

type Method string

const (
	MethodGetAppInfo          Method = "GetAppInfo"
	MethodGetDevices          Method = "GetDevices"
	MethodStartCapture        Method = "StartCapture"
	MethodLoadCapture         Method = "LoadCapture"
	MethodSaveCapture         Method = "SaveCapture"
	MethodStopCapture         Method = "StopCapture"
	MethodWaitCapture         Method = "WaitCapture"
	MethodCloseCapture        Method = "CloseCapture"
	MethodAddAnalyzer         Method = "AddAnalyzer"
	MethodRemoveAnalyzer      Method = "RemoveAnalyzer"
	MethodAddHighLevelAnalyzer    Method = "AddHighLevelAnalyzer"
	MethodRemoveHighLevelAnalyzer Method = "RemoveHighLevelAnalyzer"
	MethodExportRawDataCsv    Method = "ExportRawDataCsv"
	MethodExportRawDataBinary Method = "ExportRawDataBinary"
	MethodExportDataTableCsv  Method = "ExportDataTableCsv"
)

var AllMethods = []Method{
	MethodGetAppInfo,
	MethodGetDevices,
	MethodStartCapture,
	MethodLoadCapture,
	MethodSaveCapture,
	MethodStopCapture,
	MethodWaitCapture,
	MethodCloseCapture,
	MethodAddAnalyzer,
	MethodRemoveAnalyzer,
	MethodAddHighLevelAnalyzer,
	MethodRemoveHighLevelAnalyzer,
	MethodExportRawDataCsv,
	MethodExportRawDataBinary,
	MethodExportDataTableCsv,
}

type RuntimeContext struct {
	Ctx         context.Context
	Plan        *Plan
	State       *State
	Clock       Clock
	CallN       int
	SideEffects SideEffects
}

type Server struct {
	pb.UnimplementedManagerServer

	plan        *Plan
	clock       Clock
	sideEffects SideEffects

	mu    sync.Mutex
	state State
	calls map[Method]int
}

type Option func(*Server)

func WithClock(clock Clock) Option {
	return func(server *Server) {
		if clock != nil {
			server.clock = clock
		}
	}
}

func WithSideEffects(sideEffects SideEffects) Option {
	return func(server *Server) {
		if sideEffects != nil {
			server.sideEffects = sideEffects
		}
	}
}

func NewServer(plan *Plan, opts ...Option) *Server {
	server := &Server{
		plan:        plan,
		clock:       RealClock{},
		sideEffects: NoopSideEffects{},
		calls:       make(map[Method]int),
	}
	for _, opt := range opts {
		opt(server)
	}

	server.state = newState(plan, server.clock)
	if needsFileSideEffects(plan) {
		server.sideEffects = FileSideEffects{}
	}
	return server
}

func (s *Server) exec(ctx context.Context, method Method, req any, fn func(*RuntimeContext) (any, error)) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[method]++
	callN := s.calls[method]
	if err := s.maybeFault(method, req, callN); err != nil {
		return nil, err
	}

	runtime := &RuntimeContext{
		Ctx:         ctx,
		Plan:        s.plan,
		State:       &s.state,
		Clock:       s.clock,
		CallN:       callN,
		SideEffects: s.ensureSideEffects(),
	}

	return fn(runtime)
}

func (s *Server) maybeFault(method Method, req any, callN int) error {
	for _, fault := range s.plan.Faults {
		if fault.Method != method {
			continue
		}
		if fault.NthCall != nil && *fault.NthCall != callN {
			continue
		}
		if fault.Match != nil && !fault.Match(req) {
			continue
		}
		return statusError(fault.Code, fault.Message)
	}
	return nil
}

func newState(plan *Plan, clock Clock) State {
	state := State{
		AppInfo:        plan.Fixtures.AppInfo,
		Devices:        append([]*pb.Device{}, plan.Fixtures.Devices...),
		Captures:       make(map[uint64]*CaptureState),
		Analyzers:      make(map[uint64]map[uint64]*AnalyzerState),
		HighLevelAnalyzers: make(map[uint64]map[uint64]*HighLevelAnalyzerState),
		NextCaptureID:  plan.Defaults.CaptureIDStart,
		NextAnalyzerID: plan.Defaults.AnalyzerIDStart,
	}

	var maxCaptureID uint64
	for _, capture := range plan.Fixtures.Captures {
		startedAt := capture.StartedAt
		if startedAt.IsZero() && capture.Status == CaptureStatusRunning {
			startedAt = clock.Now()
		}
		state.Captures[capture.ID] = &CaptureState{
			ID:        capture.ID,
			Status:    capture.Status,
			Origin:    capture.Origin,
			StartedAt: startedAt,
			Mode:      capture.Mode,
		}
		if capture.ID > maxCaptureID {
			maxCaptureID = capture.ID
		}
	}

	if maxCaptureID >= state.NextCaptureID {
		state.NextCaptureID = maxCaptureID + 1
	}

	return state
}

func needsFileSideEffects(plan *Plan) bool {
	if plan.Behavior.SaveCapture.WritePlaceholderFile {
		return true
	}
	if plan.Behavior.ExportRawDataCsv.WriteDigitalCSV || plan.Behavior.ExportRawDataCsv.WriteAnalogCSV {
		return true
	}
	if plan.Behavior.ExportRawDataBinary.WriteDigitalBin || plan.Behavior.ExportRawDataBinary.WriteAnalogBin {
		return true
	}
	if plan.Behavior.ExportDataTableCsv.WritePlaceholderFile {
		return true
	}
	return false
}
