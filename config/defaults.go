package config

var (
	DefaultConfigValues = ConfigValues{
		PipelineID:            "00000000-0000-0000-0000-000000000001",
		PipelineNumber:        1,
		PipelineTriggerSource: "api",
		PipelineProjectType:   "github",
	}
)

type ConfigValues struct {
	PipelineID            string
	PipelineNumber        int
	PipelineTriggerSource string
	PipelineProjectType   string
}
