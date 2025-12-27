package saleae

import (
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Plan struct {
	Version  int
	Scenario string
	Defaults DefaultsPlan
	Fixtures FixturesPlan
	Behavior BehaviorPlan
	Faults   []FaultRule
}

type DefaultsPlan struct {
	StatusOnUnknownCaptureID codes.Code
	CaptureIDStart           uint64
	WaitCapturePolicy        WaitCapturePolicy
	WaitCaptureMaxBlock      time.Duration
}

type FixturesPlan struct {
	AppInfo  *pb.AppInfo
	Devices  []*pb.Device
	Captures []CapturePlan
}

type CapturePlan struct {
	ID        uint64
	Status    CaptureStatus
	Origin    CaptureOrigin
	StartedAt time.Time
	Mode      CaptureMode
}

type BehaviorPlan struct {
	GetDevices          GetDevicesPlan
	LoadCapture         LoadCapturePlan
	SaveCapture         SaveCapturePlan
	StopCapture         StopCapturePlan
	WaitCapture         WaitCapturePlan
	CloseCapture        CloseCapturePlan
	ExportRawDataCsv    ExportRawDataCsvPlan
	ExportRawDataBinary ExportRawDataBinaryPlan
}

type GetDevicesPlan struct {
	FilterSimulationDevices bool
}

type LoadCapturePlan struct {
	RequireNonEmptyFilepath bool
	RequireFileExists       bool
	CreateCapture           CapturePlan
}

type SaveCapturePlan struct {
	RequireCaptureExists bool
	WritePlaceholderFile bool
	PlaceholderBytes     []byte
}

type StopCapturePlan struct {
	RequireCaptureExists bool
	TransitionFrom       CaptureStatus
	TransitionTo         CaptureStatus
}

type WaitCapturePlan struct {
	RequireCaptureExists               bool
	ErrorOnManualMode                  bool
	TimedCapturesCompleteAfterDuration bool
	Policy                             WaitCapturePolicy
	MaxBlock                           time.Duration
}

type CloseCapturePlan struct {
	Mode CloseCaptureMode
}

type ExportRawDataCsvPlan struct {
	RequireCaptureExists           bool
	WriteDigitalCSV                bool
	WriteAnalogCSV                 bool
	DigitalFilename                string
	AnalogFilename                 string
	IncludeRequestedChannelsInFile bool
}

type ExportRawDataBinaryPlan struct {
	RequireCaptureExists bool
	WriteDigitalBin      bool
	WriteAnalogBin       bool
	DigitalFilename      string
	AnalogFilename       string
}

type FaultRule struct {
	Method  Method
	NthCall *int
	Match   func(any) bool
	Code    codes.Code
	Message string
}

func Compile(cfg Config) (*Plan, error) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Version != 1 {
		return nil, errors.Errorf("unsupported config version %d", cfg.Version)
	}

	defaults := DefaultsPlan{
		StatusOnUnknownCaptureID: codes.InvalidArgument,
		CaptureIDStart:           1,
		WaitCapturePolicy:        WaitCaptureImmediate,
		WaitCaptureMaxBlock:      0,
	}

	if cfg.Defaults.GRPC.StatusOnUnknownCaptureID != "" {
		code, err := parseStatusCode(cfg.Defaults.GRPC.StatusOnUnknownCaptureID)
		if err != nil {
			return nil, err
		}
		defaults.StatusOnUnknownCaptureID = code
	}
	if cfg.Defaults.IDs.CaptureIDStart != 0 {
		defaults.CaptureIDStart = cfg.Defaults.IDs.CaptureIDStart
	}
	if cfg.Defaults.Timing.WaitCapturePolicy != "" {
		policy, err := parseWaitCapturePolicy(cfg.Defaults.Timing.WaitCapturePolicy)
		if err != nil {
			return nil, err
		}
		defaults.WaitCapturePolicy = policy
	}
	if cfg.Defaults.Timing.MaxBlockMs > 0 {
		defaults.WaitCaptureMaxBlock = time.Duration(cfg.Defaults.Timing.MaxBlockMs) * time.Millisecond
	}

	fixtures, err := compileFixtures(cfg.Fixtures)
	if err != nil {
		return nil, err
	}

	behavior, err := compileBehavior(cfg, defaults)
	if err != nil {
		return nil, err
	}

	faults, err := compileFaults(cfg.Faults)
	if err != nil {
		return nil, err
	}

	return &Plan{
		Version:  cfg.Version,
		Scenario: cfg.Scenario,
		Defaults: defaults,
		Fixtures: fixtures,
		Behavior: behavior,
		Faults:   faults,
	}, nil
}

func compileFixtures(cfg FixturesConfig) (FixturesPlan, error) {
	plan := FixturesPlan{}
	if cfg.AppInfo != nil {
		appInfo, err := compileAppInfo(*cfg.AppInfo)
		if err != nil {
			return FixturesPlan{}, err
		}
		plan.AppInfo = appInfo
	}

	devices := make([]*pb.Device, 0, len(cfg.Devices))
	for _, device := range cfg.Devices {
		if device.DeviceID == "" {
			return FixturesPlan{}, errors.New("fixtures.devices.device_id is required")
		}
		if device.DeviceType == "" {
			return FixturesPlan{}, errors.New("fixtures.devices.device_type is required")
		}
		deviceType, err := parseDeviceType(device.DeviceType)
		if err != nil {
			return FixturesPlan{}, err
		}
		devices = append(devices, &pb.Device{
			DeviceId:     device.DeviceID,
			DeviceType:   deviceType,
			IsSimulation: device.IsSimulation,
		})
	}
	plan.Devices = devices

	captures := make([]CapturePlan, 0, len(cfg.Captures))
	for _, capture := range cfg.Captures {
		if capture.CaptureID == 0 {
			return FixturesPlan{}, errors.New("fixtures.captures.capture_id is required")
		}
		status, err := parseCaptureStatus(capture.Status)
		if err != nil {
			return FixturesPlan{}, err
		}
		origin, err := parseCaptureOrigin(capture.Origin)
		if err != nil {
			return FixturesPlan{}, err
		}
		mode, err := parseCaptureMode(capture.Mode)
		if err != nil {
			return FixturesPlan{}, err
		}
		var startedAt time.Time
		if capture.StartedAt != "" {
			startedAt, err = time.Parse(time.RFC3339, capture.StartedAt)
			if err != nil {
				return FixturesPlan{}, errors.Wrapf(err, "parse fixtures.captures.started_at for capture %d", capture.CaptureID)
			}
		}
		captures = append(captures, CapturePlan{
			ID:        capture.CaptureID,
			Status:    status,
			Origin:    origin,
			StartedAt: startedAt,
			Mode:      mode,
		})
	}
	plan.Captures = captures

	return plan, nil
}

func compileAppInfo(cfg AppInfoConfig) (*pb.AppInfo, error) {
	apiVersion := &pb.Version{Major: 1, Minor: 0, Patch: 0}
	if cfg.APIVersion != nil {
		apiVersion = &pb.Version{
			Major: uint32(cfg.APIVersion.Major),
			Minor: uint32(cfg.APIVersion.Minor),
			Patch: uint32(cfg.APIVersion.Patch),
		}
	}

	appVersion := cfg.ApplicationVersion
	if appVersion == "" {
		appVersion = "mock"
	}
	launchPID := cfg.LaunchPID
	if launchPID == 0 {
		launchPID = uint64(os.Getpid())
	}

	return &pb.AppInfo{
		ApiVersion:         apiVersion,
		ApplicationVersion: appVersion,
		LaunchPid:          launchPID,
	}, nil
}

func compileBehavior(cfg Config, defaults DefaultsPlan) (BehaviorPlan, error) {
	behavior := BehaviorPlan{
		GetDevices: GetDevicesPlan{
			FilterSimulationDevices: pickBool(cfg.Behavior.GetDevices.FilterSimulationDevices, true),
		},
		LoadCapture: LoadCapturePlan{
			RequireNonEmptyFilepath: pickBool(cfg.Behavior.LoadCapture.Validate.RequireNonEmptyFilepath, true),
			RequireFileExists:       pickBool(cfg.Behavior.LoadCapture.Validate.RequireFileExists, false),
			CreateCapture: CapturePlan{
				Status:    CaptureStatusCompleted,
				Origin:    CaptureOriginLoaded,
				StartedAt: time.Time{},
				Mode:      CaptureMode{Kind: CaptureModeTimed, Duration: 0},
			},
		},
		SaveCapture: SaveCapturePlan{
			RequireCaptureExists: pickBool(cfg.Behavior.SaveCapture.Validate.RequireCaptureExists, true),
			WritePlaceholderFile: pickBool(cfg.Behavior.SaveCapture.SideEffect.WritePlaceholderFile, false),
			PlaceholderBytes:     []byte(cfg.Behavior.SaveCapture.SideEffect.PlaceholderBytes),
		},
		StopCapture: StopCapturePlan{
			RequireCaptureExists: pickBool(cfg.Behavior.StopCapture.Validate.RequireCaptureExists, true),
			TransitionFrom:       CaptureStatusRunning,
			TransitionTo:         CaptureStatusStopped,
		},
		WaitCapture: WaitCapturePlan{
			RequireCaptureExists:               pickBool(cfg.Behavior.WaitCapture.Validate.RequireCaptureExists, true),
			ErrorOnManualMode:                  pickBool(cfg.Behavior.WaitCapture.Validate.ErrorOnManualMode, true),
			TimedCapturesCompleteAfterDuration: pickBool(cfg.Behavior.WaitCapture.Completion.TimedCapturesCompleteAfterDuration, true),
			Policy:                             defaults.WaitCapturePolicy,
			MaxBlock:                           defaults.WaitCaptureMaxBlock,
		},
		CloseCapture: CloseCapturePlan{
			Mode: CloseCaptureDelete,
		},
		ExportRawDataCsv: ExportRawDataCsvPlan{
			RequireCaptureExists:           pickBool(cfg.Behavior.ExportRawDataCsv.Validate.RequireCaptureExists, true),
			DigitalFilename:                "digital.csv",
			AnalogFilename:                 "analog.csv",
			IncludeRequestedChannelsInFile: pickBool(cfg.Behavior.ExportRawDataCsv.SideEffect.IncludeRequestedChannelsInFile, false),
		},
		ExportRawDataBinary: ExportRawDataBinaryPlan{
			RequireCaptureExists: pickBool(cfg.Behavior.ExportRawDataBinary.Validate.RequireCaptureExists, true),
			DigitalFilename:      "digital.bin",
			AnalogFilename:       "analog.bin",
		},
	}

	if cfg.Behavior.LoadCapture.OnCall.CreateCapture != nil {
		create := cfg.Behavior.LoadCapture.OnCall.CreateCapture
		status, err := parseCaptureStatus(create.Status)
		if err != nil {
			return BehaviorPlan{}, err
		}
		mode, err := parseCaptureMode(create.Mode)
		if err != nil {
			return BehaviorPlan{}, err
		}
		behavior.LoadCapture.CreateCapture.Status = status
		behavior.LoadCapture.CreateCapture.Mode = mode
	}

	if cfg.Behavior.SaveCapture.SideEffect.PlaceholderBytes == "" {
		behavior.SaveCapture.PlaceholderBytes = []byte("SALAD_MOCK_SAL_V1\n")
	}

	if cfg.Behavior.StopCapture.Transition.From != "" || cfg.Behavior.StopCapture.Transition.To != "" {
		from, err := parseCaptureStatus(cfg.Behavior.StopCapture.Transition.From)
		if err != nil {
			return BehaviorPlan{}, err
		}
		to, err := parseCaptureStatus(cfg.Behavior.StopCapture.Transition.To)
		if err != nil {
			return BehaviorPlan{}, err
		}
		behavior.StopCapture.TransitionFrom = from
		behavior.StopCapture.TransitionTo = to
	}

	if cfg.Behavior.WaitCapture.Validate.ErrorOnManualMode != nil {
		behavior.WaitCapture.ErrorOnManualMode = *cfg.Behavior.WaitCapture.Validate.ErrorOnManualMode
	}
	if cfg.Behavior.WaitCapture.Completion.TimedCapturesCompleteAfterDuration != nil {
		behavior.WaitCapture.TimedCapturesCompleteAfterDuration = *cfg.Behavior.WaitCapture.Completion.TimedCapturesCompleteAfterDuration
	}

	if cfg.Behavior.CloseCapture.Mode != "" {
		mode, err := parseCloseCaptureMode(cfg.Behavior.CloseCapture.Mode)
		if err != nil {
			return BehaviorPlan{}, err
		}
		behavior.CloseCapture.Mode = mode
	}

	if cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders != nil {
		behavior.ExportRawDataCsv.WriteDigitalCSV = cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.DigitalCSV
		behavior.ExportRawDataCsv.WriteAnalogCSV = cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.AnalogCSV
		if cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.Filenames != nil {
			if cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.Filenames.Digital != "" {
				behavior.ExportRawDataCsv.DigitalFilename = cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.Filenames.Digital
			}
			if cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.Filenames.Analog != "" {
				behavior.ExportRawDataCsv.AnalogFilename = cfg.Behavior.ExportRawDataCsv.SideEffect.WritePlaceholders.Filenames.Analog
			}
		}
	}

	if cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders != nil {
		behavior.ExportRawDataBinary.WriteDigitalBin = cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.DigitalBin
		behavior.ExportRawDataBinary.WriteAnalogBin = cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.AnalogBin
		if cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.Filenames != nil {
			if cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.Filenames.Digital != "" {
				behavior.ExportRawDataBinary.DigitalFilename = cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.Filenames.Digital
			}
			if cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.Filenames.Analog != "" {
				behavior.ExportRawDataBinary.AnalogFilename = cfg.Behavior.ExportRawDataBinary.SideEffect.WritePlaceholders.Filenames.Analog
			}
		}
	}

	return behavior, nil
}

func compileFaults(cfg []FaultRuleConfig) ([]FaultRule, error) {
	if len(cfg) == 0 {
		return nil, nil
	}
	faults := make([]FaultRule, 0, len(cfg))
	for _, fault := range cfg {
		if fault.When.Method == "" {
			return nil, errors.New("faults.when.method is required")
		}
		method, err := parseMethod(fault.When.Method)
		if err != nil {
			return nil, err
		}
		if fault.Respond.Status == "" {
			return nil, errors.New("faults.respond.status is required")
		}
		code, err := parseStatusCode(fault.Respond.Status)
		if err != nil {
			return nil, err
		}
		if fault.Respond.Message == "" {
			return nil, errors.New("faults.respond.message is required")
		}

		matcher, err := compileFaultMatcher(method, fault.When.Match)
		if err != nil {
			return nil, err
		}

		faults = append(faults, FaultRule{
			Method:  method,
			NthCall: fault.When.NthCall,
			Match:   matcher,
			Code:    code,
			Message: fault.Respond.Message,
		})
	}
	return faults, nil
}

func compileFaultMatcher(method Method, match *FaultMatchConfig) (func(any) bool, error) {
	if match == nil {
		return nil, nil
	}

	switch method {
	case MethodLoadCapture:
		if match.Filepath == nil {
			return nil, nil
		}
		want := *match.Filepath
		return func(req any) bool {
			reqTyped, ok := req.(*pb.LoadCaptureRequest)
			if !ok {
				return false
			}
			return reqTyped.GetFilepath() == want
		}, nil
	case MethodSaveCapture:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.SaveCaptureRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	case MethodStopCapture:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.StopCaptureRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	case MethodWaitCapture:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.WaitCaptureRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	case MethodCloseCapture:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.CloseCaptureRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	case MethodExportRawDataCsv:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.ExportRawDataCsvRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	case MethodExportRawDataBinary:
		if match.CaptureID == nil {
			return nil, nil
		}
		want := *match.CaptureID
		return func(req any) bool {
			reqTyped, ok := req.(*pb.ExportRawDataBinaryRequest)
			if !ok {
				return false
			}
			return reqTyped.GetCaptureId() == want
		}, nil
	default:
		return nil, errors.Errorf("fault matchers not supported for method %s", method)
	}
}

func parseStatusCode(code string) (codes.Code, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	parsed, ok := grpcCodeMap[code]
	if ok {
		return parsed, nil
	}
	return codes.InvalidArgument, errors.Errorf("unknown grpc status code %q", code)
}

var grpcCodeMap = map[string]codes.Code{
	"OK":                  codes.OK,
	"CANCELED":            codes.Canceled,
	"UNKNOWN":             codes.Unknown,
	"INVALID_ARGUMENT":    codes.InvalidArgument,
	"DEADLINE_EXCEEDED":   codes.DeadlineExceeded,
	"NOT_FOUND":           codes.NotFound,
	"ALREADY_EXISTS":      codes.AlreadyExists,
	"PERMISSION_DENIED":   codes.PermissionDenied,
	"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
	"FAILED_PRECONDITION": codes.FailedPrecondition,
	"ABORTED":             codes.Aborted,
	"OUT_OF_RANGE":        codes.OutOfRange,
	"UNIMPLEMENTED":       codes.Unimplemented,
	"INTERNAL":            codes.Internal,
	"UNAVAILABLE":         codes.Unavailable,
	"DATA_LOSS":           codes.DataLoss,
	"UNAUTHENTICATED":     codes.Unauthenticated,
}

func parseDeviceType(deviceType string) (pb.DeviceType, error) {
	deviceType = strings.TrimSpace(deviceType)
	value, ok := pb.DeviceType_value[deviceType]
	if !ok {
		return pb.DeviceType_DEVICE_TYPE_UNSPECIFIED, errors.Errorf("unknown device type %q", deviceType)
	}
	return pb.DeviceType(value), nil
}

func parseCaptureStatus(status string) (CaptureStatus, error) {
	if status == "" {
		return CaptureStatusCompleted, nil
	}
	switch strings.ToLower(status) {
	case "running":
		return CaptureStatusRunning, nil
	case "stopped":
		return CaptureStatusStopped, nil
	case "completed":
		return CaptureStatusCompleted, nil
	case "closed":
		return CaptureStatusClosed, nil
	default:
		return CaptureStatusCompleted, errors.Errorf("unknown capture status %q", status)
	}
}

func parseCaptureOrigin(origin string) (CaptureOrigin, error) {
	if origin == "" {
		return CaptureOriginLoaded, nil
	}
	switch strings.ToLower(origin) {
	case "loaded":
		return CaptureOriginLoaded, nil
	case "started":
		return CaptureOriginStarted, nil
	default:
		return CaptureOriginLoaded, errors.Errorf("unknown capture origin %q", origin)
	}
}

func parseCaptureMode(mode *CaptureModeConfig) (CaptureMode, error) {
	if mode == nil {
		return CaptureMode{Kind: CaptureModeTimed, Duration: 0}, nil
	}

	switch strings.ToLower(mode.Kind) {
	case "", "timed":
		return CaptureMode{Kind: CaptureModeTimed, Duration: time.Duration(mode.DurationSeconds * float64(time.Second))}, nil
	case "manual":
		return CaptureMode{Kind: CaptureModeManual, Duration: 0}, nil
	case "trigger", "digital_trigger":
		return CaptureMode{Kind: CaptureModeTrigger, Duration: 0}, nil
	default:
		return CaptureMode{}, errors.Errorf("unknown capture mode kind %q", mode.Kind)
	}
}

func parseWaitCapturePolicy(policy string) (WaitCapturePolicy, error) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", "immediate":
		return WaitCaptureImmediate, nil
	case "error_if_running":
		return WaitCaptureErrorIfRunning, nil
	case "block_until_done":
		return WaitCaptureBlockUntilDone, nil
	default:
		return WaitCaptureImmediate, errors.Errorf("unknown wait_capture_policy %q", policy)
	}
}

func parseCloseCaptureMode(mode string) (CloseCaptureMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "delete":
		return CloseCaptureDelete, nil
	case "mark_closed":
		return CloseCaptureMarkClosed, nil
	default:
		return CloseCaptureDelete, errors.Errorf("unknown close_capture mode %q", mode)
	}
}

func parseMethod(method string) (Method, error) {
	method = strings.TrimSpace(method)
	for _, candidate := range AllMethods {
		if string(candidate) == method {
			return candidate, nil
		}
	}
	return Method(""), errors.Errorf("unknown method %q", method)
}

func pickBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func statusError(code codes.Code, message string) error {
	if code == codes.OK {
		return nil
	}
	return status.Error(code, message)
}

func captureStatusError(code codes.Code, captureID uint64) error {
	return statusError(code, fmt.Sprintf("capture %d not found", captureID))
}
