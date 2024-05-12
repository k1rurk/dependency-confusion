package models

type GauStruct struct {
	Domain  string `json:"domain"`
	Threads uint   `json:"threads"`
	Timeout uint    `json:"timeout"`
	Retries uint    `json:"retries"`
}
