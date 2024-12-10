package system

type CPUSimple struct {
	UsedPercent string `json:"usedPercent"`
	Used        string `json:"used"`
	Cores       int    `json:"cores"`
}
