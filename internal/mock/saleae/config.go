package saleae

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int               `yaml:"version"`
	Scenario string            `yaml:"scenario,omitempty"`
	Defaults DefaultsConfig    `yaml:"defaults,omitempty"`
	Fixtures FixturesConfig    `yaml:"fixtures,omitempty"`
	Behavior BehaviorConfig    `yaml:"behavior,omitempty"`
	Faults   []FaultRuleConfig `yaml:"faults,omitempty"`
}

type DefaultsConfig struct {
	GRPC   GRPCDefaultsConfig   `yaml:"grpc,omitempty"`
	IDs    IDsDefaultsConfig    `yaml:"ids,omitempty"`
	Timing TimingDefaultsConfig `yaml:"timing,omitempty"`
}

type GRPCDefaultsConfig struct {
	StatusOnUnknownCaptureID string `yaml:"status_on_unknown_capture_id,omitempty"`
}

type IDsDefaultsConfig struct {
	CaptureIDStart  uint64 `yaml:"capture_id_start,omitempty"`
	AnalyzerIDStart uint64 `yaml:"analyzer_id_start,omitempty"`
	Deterministic   *bool  `yaml:"deterministic,omitempty"`
}

type TimingDefaultsConfig struct {
	WaitCapturePolicy string `yaml:"wait_capture_policy,omitempty"`
	MaxBlockMs        int    `yaml:"max_block_ms,omitempty"`
}

type FixturesConfig struct {
	AppInfo  *AppInfoConfig   `yaml:"appinfo,omitempty"`
	Devices  []DeviceConfig   `yaml:"devices,omitempty"`
	Captures []CaptureFixture `yaml:"captures,omitempty"`
}

type AppInfoConfig struct {
	ApplicationVersion string         `yaml:"application_version,omitempty"`
	APIVersion         *VersionConfig `yaml:"api_version,omitempty"`
	LaunchPID          uint64         `yaml:"launch_pid,omitempty"`
}

type VersionConfig struct {
	Major int32 `yaml:"major,omitempty"`
	Minor int32 `yaml:"minor,omitempty"`
	Patch int32 `yaml:"patch,omitempty"`
}

type DeviceConfig struct {
	DeviceID     string `yaml:"device_id,omitempty"`
	DeviceType   string `yaml:"device_type,omitempty"`
	IsSimulation bool   `yaml:"is_simulation,omitempty"`
}

type CaptureFixture struct {
	CaptureID uint64             `yaml:"capture_id,omitempty"`
	Status    string             `yaml:"status,omitempty"`
	Origin    string             `yaml:"origin,omitempty"`
	StartedAt string             `yaml:"started_at,omitempty"`
	Mode      *CaptureModeConfig `yaml:"mode,omitempty"`
}

type CaptureModeConfig struct {
	Kind            string  `yaml:"kind,omitempty"`
	DurationSeconds float64 `yaml:"duration_seconds,omitempty"`
}

type BehaviorConfig struct {
	GetDevices          GetDevicesBehaviorConfig          `yaml:"GetDevices,omitempty"`
	StartCapture        StartCaptureBehaviorConfig        `yaml:"StartCapture,omitempty"`
	LoadCapture         LoadCaptureBehaviorConfig         `yaml:"LoadCapture,omitempty"`
	SaveCapture         SaveCaptureBehaviorConfig         `yaml:"SaveCapture,omitempty"`
	StopCapture         StopCaptureBehaviorConfig         `yaml:"StopCapture,omitempty"`
	WaitCapture         WaitCaptureBehaviorConfig         `yaml:"WaitCapture,omitempty"`
	CloseCapture        CloseCaptureBehaviorConfig        `yaml:"CloseCapture,omitempty"`
	AddAnalyzer         AddAnalyzerBehaviorConfig         `yaml:"AddAnalyzer,omitempty"`
	RemoveAnalyzer      RemoveAnalyzerBehaviorConfig      `yaml:"RemoveAnalyzer,omitempty"`
	ExportRawDataCsv    ExportRawDataCsvBehaviorConfig    `yaml:"ExportRawDataCsv,omitempty"`
	ExportRawDataBinary ExportRawDataBinaryBehaviorConfig `yaml:"ExportRawDataBinary,omitempty"`
}

type GetDevicesBehaviorConfig struct {
	FilterSimulationDevices *bool `yaml:"filter_simulation_devices,omitempty"`
}

type StartCaptureBehaviorConfig struct {
	Validate StartCaptureValidateConfig `yaml:"validate,omitempty"`
	OnCall   StartCaptureOnCallConfig   `yaml:"on_call,omitempty"`
}

type StartCaptureValidateConfig struct {
	RequireDeviceExists *bool `yaml:"require_device_exists,omitempty"`
}

type StartCaptureOnCallConfig struct {
	CreateCapture *CaptureCreateConfig `yaml:"create_capture,omitempty"`
}

type AddAnalyzerBehaviorConfig struct {
	Validate AddAnalyzerValidateConfig `yaml:"validate,omitempty"`
}

type AddAnalyzerValidateConfig struct {
	RequireCaptureExists        *bool `yaml:"require_capture_exists,omitempty"`
	RequireAnalyzerNameNonEmpty *bool `yaml:"require_analyzer_name_non_empty,omitempty"`
}

type RemoveAnalyzerBehaviorConfig struct {
	Validate RemoveAnalyzerValidateConfig `yaml:"validate,omitempty"`
}

type RemoveAnalyzerValidateConfig struct {
	RequireCaptureExists  *bool `yaml:"require_capture_exists,omitempty"`
	RequireAnalyzerExists *bool `yaml:"require_analyzer_exists,omitempty"`
}

type LoadCaptureBehaviorConfig struct {
	Validate LoadCaptureValidateConfig `yaml:"validate,omitempty"`
	OnCall   LoadCaptureOnCallConfig   `yaml:"on_call,omitempty"`
}

type LoadCaptureValidateConfig struct {
	RequireNonEmptyFilepath *bool `yaml:"require_non_empty_filepath,omitempty"`
	RequireFileExists       *bool `yaml:"require_file_exists,omitempty"`
}

type LoadCaptureOnCallConfig struct {
	CreateCapture *CaptureCreateConfig `yaml:"create_capture,omitempty"`
}

type CaptureCreateConfig struct {
	Status string             `yaml:"status,omitempty"`
	Mode   *CaptureModeConfig `yaml:"mode,omitempty"`
}

type SaveCaptureBehaviorConfig struct {
	Validate   SaveCaptureValidateConfig   `yaml:"validate,omitempty"`
	SideEffect SaveCaptureSideEffectConfig `yaml:"side_effect,omitempty"`
}

type SaveCaptureValidateConfig struct {
	RequireCaptureExists *bool `yaml:"require_capture_exists,omitempty"`
}

type SaveCaptureSideEffectConfig struct {
	WritePlaceholderFile *bool  `yaml:"write_placeholder_file,omitempty"`
	PlaceholderBytes     string `yaml:"placeholder_bytes,omitempty"`
}

type StopCaptureBehaviorConfig struct {
	Validate   StopCaptureValidateConfig `yaml:"validate,omitempty"`
	Transition TransitionConfig          `yaml:"transition,omitempty"`
}

type StopCaptureValidateConfig struct {
	RequireCaptureExists *bool `yaml:"require_capture_exists,omitempty"`
}

type TransitionConfig struct {
	From string `yaml:"from,omitempty"`
	To   string `yaml:"to,omitempty"`
}

type WaitCaptureBehaviorConfig struct {
	Validate   WaitCaptureValidateConfig   `yaml:"validate,omitempty"`
	Completion WaitCaptureCompletionConfig `yaml:"completion,omitempty"`
}

type WaitCaptureValidateConfig struct {
	RequireCaptureExists *bool `yaml:"require_capture_exists,omitempty"`
	ErrorOnManualMode    *bool `yaml:"error_on_manual_mode,omitempty"`
}

type WaitCaptureCompletionConfig struct {
	TimedCapturesCompleteAfterDuration *bool `yaml:"timed_captures_complete_after_duration,omitempty"`
}

type CloseCaptureBehaviorConfig struct {
	Mode string `yaml:"mode,omitempty"`
}

type ExportRawDataCsvBehaviorConfig struct {
	Validate   ExportValidateConfig   `yaml:"validate,omitempty"`
	SideEffect ExportRawCsvSideEffect `yaml:"side_effect,omitempty"`
}

type ExportRawDataBinaryBehaviorConfig struct {
	Validate   ExportValidateConfig      `yaml:"validate,omitempty"`
	SideEffect ExportRawBinarySideEffect `yaml:"side_effect,omitempty"`
}

type ExportValidateConfig struct {
	RequireCaptureExists *bool `yaml:"require_capture_exists,omitempty"`
}

type ExportRawCsvSideEffect struct {
	WritePlaceholders              *ExportRawCsvPlaceholders `yaml:"write_placeholders,omitempty"`
	IncludeRequestedChannelsInFile *bool                     `yaml:"include_requested_channels_in_file,omitempty"`
}

type ExportRawCsvPlaceholders struct {
	DigitalCSV bool                   `yaml:"digital_csv,omitempty"`
	AnalogCSV  bool                   `yaml:"analog_csv,omitempty"`
	Filenames  *ExportRawCsvFilenames `yaml:"filenames,omitempty"`
}

type ExportRawCsvFilenames struct {
	Digital string `yaml:"digital,omitempty"`
	Analog  string `yaml:"analog,omitempty"`
}

type ExportRawBinarySideEffect struct {
	WritePlaceholders *ExportRawBinaryPlaceholders `yaml:"write_placeholders,omitempty"`
}

type ExportRawBinaryPlaceholders struct {
	DigitalBin bool                      `yaml:"digital_bin,omitempty"`
	AnalogBin  bool                      `yaml:"analog_bin,omitempty"`
	Filenames  *ExportRawBinaryFilenames `yaml:"filenames,omitempty"`
}

type ExportRawBinaryFilenames struct {
	Digital string `yaml:"digital,omitempty"`
	Analog  string `yaml:"analog,omitempty"`
}

type FaultRuleConfig struct {
	When    FaultWhenConfig    `yaml:"when,omitempty"`
	Respond FaultRespondConfig `yaml:"respond,omitempty"`
}

type FaultWhenConfig struct {
	Method  string            `yaml:"method,omitempty"`
	NthCall *int              `yaml:"nth_call,omitempty"`
	Match   *FaultMatchConfig `yaml:"match,omitempty"`
}

type FaultMatchConfig struct {
	CaptureID    *uint64 `yaml:"capture_id,omitempty"`
	Filepath     *string `yaml:"filepath,omitempty"`
	AnalyzerID   *uint64 `yaml:"analyzer_id,omitempty"`
	AnalyzerName *string `yaml:"analyzer_name,omitempty"`
}

type FaultRespondConfig struct {
	Status  string `yaml:"status,omitempty"`
	Message string `yaml:"message,omitempty"`
}

func LoadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, errors.Wrapf(err, "open mock config %s", path)
	}
	defer func() { _ = file.Close() }()

	return LoadConfigFromReader(file)
}

func LoadConfigFromReader(reader io.Reader) (Config, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "decode mock config")
	}
	return cfg, nil
}
