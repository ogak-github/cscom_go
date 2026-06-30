package metrics

import (
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
)

type Network struct {
	Name    string  `json:"name"`
	RXBytes uint64  `json:"rx_bytes"`
	TXBytes uint64  `json:"tx_bytes"`
	RXRate  float64 `json:"rx_rate"`
	TXRate  float64 `json:"tx_rate"`
}

type netSample struct {
	counters []gnet.IOCountersStat
	ts       time.Time
}

func isVirtualIface(name string) bool {
	for _, prefix := range []string{"lo", "veth", "br-", "virbr", "docker", "vnet", "tun", "tap"} {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func collectNetwork(prev *netSample) ([]Network, *netSample, error) {
	counters, err := gnet.IOCounters(true)
	if err != nil {
		return nil, prev, err
	}
	cur := &netSample{counters: counters, ts: time.Now()}

	var nets []Network
	for _, c := range counters {
		if isVirtualIface(c.Name) {
			continue
		}
		n := Network{
			Name:    c.Name,
			RXBytes: c.BytesRecv,
			TXBytes: c.BytesSent,
		}
		if prev != nil {
			dt := cur.ts.Sub(prev.ts).Seconds()
			if dt > 0 {
				for _, p := range prev.counters {
					if p.Name == c.Name {
						if rx := int64(c.BytesRecv) - int64(p.BytesRecv); rx >= 0 {
							n.RXRate = float64(rx) / dt
						}
						if tx := int64(c.BytesSent) - int64(p.BytesSent); tx >= 0 {
							n.TXRate = float64(tx) / dt
						}
						break
					}
				}
			}
		}
		nets = append(nets, n)
	}

	return nets, cur, nil
}
