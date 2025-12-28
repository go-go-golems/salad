package saleae

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) Register(grpcServer *grpc.Server) {
	pb.RegisterManagerServer(grpcServer, s)
}

func (s *Server) Start(addr string) (*grpc.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "listen on %s", addr)
	}

	grpcServer := grpc.NewServer()
	s.Register(grpcServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	return grpcServer, listener, nil
}

func (s *Server) GetAppInfo(ctx context.Context, req *pb.GetAppInfoRequest) (*pb.GetAppInfoReply, error) {
	out, err := s.exec(ctx, MethodGetAppInfo, req, func(runtime *RuntimeContext) (any, error) {
		if runtime.State.AppInfo == nil {
			return &pb.GetAppInfoReply{AppInfo: &pb.AppInfo{}}, nil
		}
		return &pb.GetAppInfoReply{AppInfo: runtime.State.AppInfo}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.GetAppInfoReply), nil
}

func (s *Server) GetDevices(ctx context.Context, req *pb.GetDevicesRequest) (*pb.GetDevicesReply, error) {
	out, err := s.exec(ctx, MethodGetDevices, req, func(runtime *RuntimeContext) (any, error) {
		devices := runtime.State.Devices
		if runtime.Plan.Behavior.GetDevices.FilterSimulationDevices && !req.GetIncludeSimulationDevices() {
			filtered := make([]*pb.Device, 0, len(devices))
			for _, device := range devices {
				if device.GetIsSimulation() {
					continue
				}
				filtered = append(filtered, device)
			}
			devices = filtered
		}
		return &pb.GetDevicesReply{Devices: devices}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.GetDevicesReply), nil
}

func (s *Server) StartCapture(ctx context.Context, req *pb.StartCaptureRequest) (*pb.StartCaptureReply, error) {
	out, err := s.exec(ctx, MethodStartCapture, req, func(runtime *RuntimeContext) (any, error) {
		// Device selection: empty device_id → first physical device
		deviceID := req.GetDeviceId()
		if deviceID == "" {
			for _, device := range runtime.State.Devices {
				if !device.GetIsSimulation() {
					deviceID = device.GetDeviceId()
					break
				}
			}
			if deviceID == "" {
				return nil, status.Error(codes.NotFound, "StartCapture: no physical device available")
			}
		}

		// Validate device exists
		if runtime.Plan.Behavior.StartCapture.RequireDeviceExists {
			deviceExists := false
			for _, device := range runtime.State.Devices {
				if device.GetDeviceId() == deviceID {
					deviceExists = true
					break
				}
			}
			if !deviceExists {
				return nil, status.Error(codes.NotFound, fmt.Sprintf("StartCapture: device %q not found", deviceID))
			}
		}

		// Extract capture mode from CaptureConfiguration
		captureConfig := req.GetCaptureConfiguration()
		if captureConfig == nil {
			return nil, status.Error(codes.InvalidArgument, "StartCapture: capture_configuration is required")
		}

		var captureMode CaptureMode
		if manualMode := captureConfig.GetManualCaptureMode(); manualMode != nil {
			captureMode = CaptureMode{Kind: CaptureModeManual, Duration: 0}
		} else if timedMode := captureConfig.GetTimedCaptureMode(); timedMode != nil {
			duration := time.Duration(timedMode.GetDurationSeconds() * float64(time.Second))
			captureMode = CaptureMode{Kind: CaptureModeTimed, Duration: duration}
		} else if triggerMode := captureConfig.GetDigitalCaptureMode(); triggerMode != nil {
			captureMode = CaptureMode{Kind: CaptureModeTrigger, Duration: 0}
		} else {
			// Default to manual if no mode specified
			captureMode = CaptureMode{Kind: CaptureModeManual, Duration: 0}
		}

		// Create capture state
		captureID := runtime.State.NextCaptureID
		runtime.State.NextCaptureID++

		capturePlan := runtime.Plan.Behavior.StartCapture.CreateCapture
		capture := &CaptureState{
			ID:        captureID,
			Status:    capturePlan.Status,
			Origin:    CaptureOriginStarted,
			StartedAt: runtime.Clock.Now(),
			Mode:      captureMode,
		}
		runtime.State.Captures[captureID] = capture

		return &pb.StartCaptureReply{
			CaptureInfo: &pb.CaptureInfo{CaptureId: captureID},
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.StartCaptureReply), nil
}

func (s *Server) LoadCapture(ctx context.Context, req *pb.LoadCaptureRequest) (*pb.LoadCaptureReply, error) {
	out, err := s.exec(ctx, MethodLoadCapture, req, func(runtime *RuntimeContext) (any, error) {
		if runtime.Plan.Behavior.LoadCapture.RequireNonEmptyFilepath && req.GetFilepath() == "" {
			return nil, status.Error(codes.InvalidArgument, "LoadCapture: filepath is required")
		}
		if runtime.Plan.Behavior.LoadCapture.RequireFileExists {
			if _, err := os.Stat(req.GetFilepath()); err != nil {
				return nil, status.Error(codes.InvalidArgument, "LoadCapture: file does not exist")
			}
		}

		captureID := runtime.State.NextCaptureID
		runtime.State.NextCaptureID++

		capturePlan := runtime.Plan.Behavior.LoadCapture.CreateCapture
		capture := &CaptureState{
			ID:        captureID,
			Status:    capturePlan.Status,
			Origin:    CaptureOriginLoaded,
			StartedAt: runtime.Clock.Now(),
			Mode:      capturePlan.Mode,
		}
		runtime.State.Captures[captureID] = capture

		return &pb.LoadCaptureReply{
			CaptureInfo: &pb.CaptureInfo{CaptureId: captureID},
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.LoadCaptureReply), nil
}

func (s *Server) SaveCapture(ctx context.Context, req *pb.SaveCaptureRequest) (*pb.SaveCaptureReply, error) {
	out, err := s.exec(ctx, MethodSaveCapture, req, func(runtime *RuntimeContext) (any, error) {
		capture, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID)
		if err != nil {
			if !runtime.Plan.Behavior.SaveCapture.RequireCaptureExists {
				return &pb.SaveCaptureReply{}, nil
			}
			return nil, err
		}

		if runtime.Plan.Behavior.SaveCapture.WritePlaceholderFile {
			if err := runtime.SideEffects.SaveCapture(req.GetFilepath(), capture.ID, runtime.Plan.Behavior.SaveCapture.PlaceholderBytes); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		return &pb.SaveCaptureReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.SaveCaptureReply), nil
}

func (s *Server) StopCapture(ctx context.Context, req *pb.StopCaptureRequest) (*pb.StopCaptureReply, error) {
	out, err := s.exec(ctx, MethodStopCapture, req, func(runtime *RuntimeContext) (any, error) {
		capture, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID)
		if err != nil {
			if !runtime.Plan.Behavior.StopCapture.RequireCaptureExists {
				return &pb.StopCaptureReply{}, nil
			}
			return nil, err
		}

		if capture.Status == runtime.Plan.Behavior.StopCapture.TransitionFrom {
			capture.Status = runtime.Plan.Behavior.StopCapture.TransitionTo
		}

		return &pb.StopCaptureReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.StopCaptureReply), nil
}

func (s *Server) WaitCapture(ctx context.Context, req *pb.WaitCaptureRequest) (*pb.WaitCaptureReply, error) {
	out, err := s.exec(ctx, MethodWaitCapture, req, func(runtime *RuntimeContext) (any, error) {
		capture, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID)
		if err != nil {
			if !runtime.Plan.Behavior.WaitCapture.RequireCaptureExists {
				return &pb.WaitCaptureReply{}, nil
			}
			return nil, err
		}

		if capture.Mode.Kind == CaptureModeManual && runtime.Plan.Behavior.WaitCapture.ErrorOnManualMode {
			return nil, status.Error(codes.InvalidArgument, "WaitCapture: manual capture mode does not support waiting")
		}

		if capture.Mode.Kind == CaptureModeTimed && runtime.Plan.Behavior.WaitCapture.TimedCapturesCompleteAfterDuration {
			if capture.Mode.Duration <= 0 {
				capture.Status = CaptureStatusCompleted
			} else if capture.StartedAt.Add(capture.Mode.Duration).Before(runtime.Clock.Now()) {
				capture.Status = CaptureStatusCompleted
			}
		}

		if capture.Status == CaptureStatusCompleted {
			return &pb.WaitCaptureReply{}, nil
		}

		switch runtime.Plan.Behavior.WaitCapture.Policy {
		case WaitCaptureImmediate:
			return nil, status.Error(codes.DeadlineExceeded, "WaitCapture: capture still running")
		case WaitCaptureErrorIfRunning:
			if capture.Status == CaptureStatusRunning {
				return nil, status.Error(codes.DeadlineExceeded, "WaitCapture: capture still running")
			}
			return &pb.WaitCaptureReply{}, nil
		case WaitCaptureBlockUntilDone:
			return runtime.blockUntilDone(capture)
		default:
			return nil, status.Error(codes.Internal, "WaitCapture: unknown policy")
		}
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.WaitCaptureReply), nil
}

func (s *Server) CloseCapture(ctx context.Context, req *pb.CloseCaptureRequest) (*pb.CloseCaptureReply, error) {
	out, err := s.exec(ctx, MethodCloseCapture, req, func(runtime *RuntimeContext) (any, error) {
		captureID := req.GetCaptureId()
		capture, err := runtime.State.captureFor(captureID, runtime.Plan.Defaults.StatusOnUnknownCaptureID)
		if err != nil {
			return nil, err
		}

		switch runtime.Plan.Behavior.CloseCapture.Mode {
		case CloseCaptureDelete:
			// When deleting a capture, also delete analyzers attached to it to avoid state leaks.
			delete(runtime.State.Analyzers, captureID)
			delete(runtime.State.HighLevelAnalyzers, captureID)
			delete(runtime.State.Captures, captureID)
		case CloseCaptureMarkClosed:
			capture.Status = CaptureStatusClosed
		}

		return &pb.CloseCaptureReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.CloseCaptureReply), nil
}

func (s *Server) ExportRawDataCsv(ctx context.Context, req *pb.ExportRawDataCsvRequest) (*pb.ExportRawDataCsvReply, error) {
	out, err := s.exec(ctx, MethodExportRawDataCsv, req, func(runtime *RuntimeContext) (any, error) {
		if runtime.Plan.Behavior.ExportRawDataCsv.RequireCaptureExists {
			if _, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		if runtime.Plan.Behavior.ExportRawDataCsv.WriteDigitalCSV || runtime.Plan.Behavior.ExportRawDataCsv.WriteAnalogCSV {
			err := runtime.SideEffects.ExportRawCSV(req.GetDirectory(), req, ExportCSVOptions{
				WriteDigital:             runtime.Plan.Behavior.ExportRawDataCsv.WriteDigitalCSV,
				WriteAnalog:              runtime.Plan.Behavior.ExportRawDataCsv.WriteAnalogCSV,
				DigitalFilename:          runtime.Plan.Behavior.ExportRawDataCsv.DigitalFilename,
				AnalogFilename:           runtime.Plan.Behavior.ExportRawDataCsv.AnalogFilename,
				IncludeRequestedChannels: runtime.Plan.Behavior.ExportRawDataCsv.IncludeRequestedChannelsInFile,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		return &pb.ExportRawDataCsvReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.ExportRawDataCsvReply), nil
}

func (s *Server) ExportRawDataBinary(ctx context.Context, req *pb.ExportRawDataBinaryRequest) (*pb.ExportRawDataBinaryReply, error) {
	out, err := s.exec(ctx, MethodExportRawDataBinary, req, func(runtime *RuntimeContext) (any, error) {
		if runtime.Plan.Behavior.ExportRawDataBinary.RequireCaptureExists {
			if _, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		if runtime.Plan.Behavior.ExportRawDataBinary.WriteDigitalBin || runtime.Plan.Behavior.ExportRawDataBinary.WriteAnalogBin {
			err := runtime.SideEffects.ExportRawBinary(req.GetDirectory(), req, ExportBinaryOptions{
				WriteDigital:    runtime.Plan.Behavior.ExportRawDataBinary.WriteDigitalBin,
				WriteAnalog:     runtime.Plan.Behavior.ExportRawDataBinary.WriteAnalogBin,
				DigitalFilename: runtime.Plan.Behavior.ExportRawDataBinary.DigitalFilename,
				AnalogFilename:  runtime.Plan.Behavior.ExportRawDataBinary.AnalogFilename,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		return &pb.ExportRawDataBinaryReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.ExportRawDataBinaryReply), nil
}

func (s *Server) ExportDataTableCsv(ctx context.Context, req *pb.ExportDataTableCsvRequest) (*pb.ExportDataTableCsvReply, error) {
	out, err := s.exec(ctx, MethodExportDataTableCsv, req, func(runtime *RuntimeContext) (any, error) {
		if runtime.Plan.Behavior.ExportDataTableCsv.RequireCaptureExists {
			if _, err := runtime.State.captureFor(req.GetCaptureId(), runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		if runtime.Plan.Behavior.ExportDataTableCsv.WritePlaceholderFile {
			err := runtime.SideEffects.ExportDataTableCSV(req.GetFilepath(), req, ExportDataTableCSVOptions{
				IncludeRequest: runtime.Plan.Behavior.ExportDataTableCsv.IncludeRequestInFile,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		return &pb.ExportDataTableCsvReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.ExportDataTableCsvReply), nil
}

func (s *Server) AddAnalyzer(ctx context.Context, req *pb.AddAnalyzerRequest) (*pb.AddAnalyzerReply, error) {
	out, err := s.exec(ctx, MethodAddAnalyzer, req, func(runtime *RuntimeContext) (any, error) {
		captureID := req.GetCaptureId()
		if captureID == 0 {
			return nil, status.Error(codes.InvalidArgument, "AddAnalyzer: capture_id is required")
		}

		if runtime.Plan.Behavior.AddAnalyzer.RequireCaptureExists {
			if _, err := runtime.State.captureFor(captureID, runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		if runtime.Plan.Behavior.AddAnalyzer.RequireAnalyzerNameNonEmpty && req.GetAnalyzerName() == "" {
			return nil, status.Error(codes.InvalidArgument, "AddAnalyzer: analyzer_name is required")
		}

		analyzerID := runtime.State.NextAnalyzerID
		runtime.State.NextAnalyzerID++

		if runtime.State.Analyzers[captureID] == nil {
			runtime.State.Analyzers[captureID] = make(map[uint64]*AnalyzerState)
		}

		settingsCopy := make(map[string]*pb.AnalyzerSettingValue, len(req.GetSettings()))
		for k, v := range req.GetSettings() {
			settingsCopy[k] = v
		}

		runtime.State.Analyzers[captureID][analyzerID] = &AnalyzerState{
			ID:        analyzerID,
			CaptureID: captureID,
			Name:      req.GetAnalyzerName(),
			Label:     req.GetAnalyzerLabel(),
			Settings:  settingsCopy,
			CreatedAt: runtime.Clock.Now(),
		}

		return &pb.AddAnalyzerReply{AnalyzerId: analyzerID}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.AddAnalyzerReply), nil
}

func (s *Server) RemoveAnalyzer(ctx context.Context, req *pb.RemoveAnalyzerRequest) (*pb.RemoveAnalyzerReply, error) {
	out, err := s.exec(ctx, MethodRemoveAnalyzer, req, func(runtime *RuntimeContext) (any, error) {
		captureID := req.GetCaptureId()
		if captureID == 0 {
			return nil, status.Error(codes.InvalidArgument, "RemoveAnalyzer: capture_id is required")
		}
		analyzerID := req.GetAnalyzerId()
		if analyzerID == 0 {
			return nil, status.Error(codes.InvalidArgument, "RemoveAnalyzer: analyzer_id is required")
		}

		if runtime.Plan.Behavior.RemoveAnalyzer.RequireCaptureExists {
			if _, err := runtime.State.captureFor(captureID, runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		byCapture := runtime.State.Analyzers[captureID]
		if byCapture == nil {
			if runtime.Plan.Behavior.RemoveAnalyzer.RequireAnalyzerExists {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("RemoveAnalyzer: analyzer %d not found", analyzerID))
			}
			return &pb.RemoveAnalyzerReply{}, nil
		}

		if _, ok := byCapture[analyzerID]; !ok {
			if runtime.Plan.Behavior.RemoveAnalyzer.RequireAnalyzerExists {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("RemoveAnalyzer: analyzer %d not found", analyzerID))
			}
			return &pb.RemoveAnalyzerReply{}, nil
		}

		delete(byCapture, analyzerID)
		return &pb.RemoveAnalyzerReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.RemoveAnalyzerReply), nil
}

func (s *Server) AddHighLevelAnalyzer(ctx context.Context, req *pb.AddHighLevelAnalyzerRequest) (*pb.AddHighLevelAnalyzerReply, error) {
	out, err := s.exec(ctx, MethodAddHighLevelAnalyzer, req, func(runtime *RuntimeContext) (any, error) {
		captureID := req.GetCaptureId()
		if captureID == 0 {
			return nil, status.Error(codes.InvalidArgument, "AddHighLevelAnalyzer: capture_id is required")
		}

		if runtime.Plan.Behavior.AddHighLevelAnalyzer.RequireCaptureExists {
			if _, err := runtime.State.captureFor(captureID, runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		if runtime.Plan.Behavior.AddHighLevelAnalyzer.RequireExtensionDirNonEmpty && req.GetExtensionDirectory() == "" {
			return nil, status.Error(codes.InvalidArgument, "AddHighLevelAnalyzer: extension_directory is required")
		}
		if runtime.Plan.Behavior.AddHighLevelAnalyzer.RequireHLANameNonEmpty && req.GetHlaName() == "" {
			return nil, status.Error(codes.InvalidArgument, "AddHighLevelAnalyzer: hla_name is required")
		}
		if runtime.Plan.Behavior.AddHighLevelAnalyzer.RequireInputAnalyzerIDNonZero && req.GetInputAnalyzerId() == 0 {
			return nil, status.Error(codes.InvalidArgument, "AddHighLevelAnalyzer: input_analyzer_id is required")
		}
		if runtime.Plan.Behavior.AddHighLevelAnalyzer.RequireInputAnalyzerExists {
			byCapture := runtime.State.Analyzers[captureID]
			if byCapture == nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("AddHighLevelAnalyzer: input analyzer %d not found", req.GetInputAnalyzerId()))
			}
			if _, ok := byCapture[req.GetInputAnalyzerId()]; !ok {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("AddHighLevelAnalyzer: input analyzer %d not found", req.GetInputAnalyzerId()))
			}
		}

		analyzerID := runtime.State.NextAnalyzerID
		runtime.State.NextAnalyzerID++

		if runtime.State.HighLevelAnalyzers[captureID] == nil {
			runtime.State.HighLevelAnalyzers[captureID] = make(map[uint64]*HighLevelAnalyzerState)
		}

		settingsCopy := make(map[string]*pb.HighLevelAnalyzerSettingValue, len(req.GetSettings()))
		for k, v := range req.GetSettings() {
			settingsCopy[k] = v
		}

		runtime.State.HighLevelAnalyzers[captureID][analyzerID] = &HighLevelAnalyzerState{
			ID:             analyzerID,
			CaptureID:       captureID,
			ExtensionDir:    req.GetExtensionDirectory(),
			HLAName:         req.GetHlaName(),
			Label:           req.GetHlaLabel(),
			InputAnalyzerID: req.GetInputAnalyzerId(),
			Settings:        settingsCopy,
			CreatedAt:       runtime.Clock.Now(),
		}

		return &pb.AddHighLevelAnalyzerReply{AnalyzerId: analyzerID}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.AddHighLevelAnalyzerReply), nil
}

func (s *Server) RemoveHighLevelAnalyzer(ctx context.Context, req *pb.RemoveHighLevelAnalyzerRequest) (*pb.RemoveHighLevelAnalyzerReply, error) {
	out, err := s.exec(ctx, MethodRemoveHighLevelAnalyzer, req, func(runtime *RuntimeContext) (any, error) {
		captureID := req.GetCaptureId()
		if captureID == 0 {
			return nil, status.Error(codes.InvalidArgument, "RemoveHighLevelAnalyzer: capture_id is required")
		}
		analyzerID := req.GetAnalyzerId()
		if analyzerID == 0 {
			return nil, status.Error(codes.InvalidArgument, "RemoveHighLevelAnalyzer: analyzer_id is required")
		}

		if runtime.Plan.Behavior.RemoveHighLevelAnalyzer.RequireCaptureExists {
			if _, err := runtime.State.captureFor(captureID, runtime.Plan.Defaults.StatusOnUnknownCaptureID); err != nil {
				return nil, err
			}
		}

		byCapture := runtime.State.HighLevelAnalyzers[captureID]
		if byCapture == nil {
			if runtime.Plan.Behavior.RemoveHighLevelAnalyzer.RequireAnalyzerExists {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("RemoveHighLevelAnalyzer: analyzer %d not found", analyzerID))
			}
			return &pb.RemoveHighLevelAnalyzerReply{}, nil
		}

		if _, ok := byCapture[analyzerID]; !ok {
			if runtime.Plan.Behavior.RemoveHighLevelAnalyzer.RequireAnalyzerExists {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("RemoveHighLevelAnalyzer: analyzer %d not found", analyzerID))
			}
			return &pb.RemoveHighLevelAnalyzerReply{}, nil
		}

		delete(byCapture, analyzerID)
		return &pb.RemoveHighLevelAnalyzerReply{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out.(*pb.RemoveHighLevelAnalyzerReply), nil
}

func (runtime *RuntimeContext) blockUntilDone(capture *CaptureState) (*pb.WaitCaptureReply, error) {
	if capture.Status == CaptureStatusCompleted {
		return &pb.WaitCaptureReply{}, nil
	}

	if capture.Mode.Kind == CaptureModeTimed && capture.Mode.Duration > 0 {
		deadline := runtime.Clock.Now().Add(runtime.Plan.Behavior.WaitCapture.MaxBlock)
		completion := capture.StartedAt.Add(capture.Mode.Duration)
		if completion.Before(deadline) || completion.Equal(deadline) {
			capture.Status = CaptureStatusCompleted
			return &pb.WaitCaptureReply{}, nil
		}
	}

	return nil, status.Error(codes.DeadlineExceeded, "WaitCapture: capture still running")
}

func (state *State) captureFor(captureID uint64, missingStatus codes.Code) (*CaptureState, error) {
	if captureID == 0 {
		return nil, status.Error(codes.InvalidArgument, "capture id is required")
	}
	capture, ok := state.Captures[captureID]
	if !ok {
		if missingStatus == codes.OK {
			return nil, status.Error(codes.InvalidArgument, "capture not found")
		}
		return nil, captureStatusError(missingStatus, captureID)
	}
	return capture, nil
}

func (s *Server) ensureSideEffects() SideEffects {
	if s.sideEffects == nil {
		return NoopSideEffects{}
	}
	return s.sideEffects
}
