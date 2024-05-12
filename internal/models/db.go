package models

import "time"

type DbPackage struct {
	Id        		int
	Timestamp 		time.Time
	SourceIP  		string
	DataExfiltrated DataExfiltrated
}

type DataExfiltrated struct {
	Hostname  string `json:"h"`
	Username  string `json:"d"`
	CWD       string `json:"c"`
    Package   string `json:"p"`
}