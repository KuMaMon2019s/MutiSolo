package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type tailscaleStatusJSON struct {
	BackendState   string `json:"BackendState"`
	CurrentTailnet struct {
		Name string `json:"Name"`
	} `json:"CurrentTailnet"`
	Self tailscaleNode            `json:"Self"`
	Peer map[string]tailscaleNode `json:"Peer"`
}

type tailscaleNode struct {
	ID           string   `json:"ID"`
	HostName     string   `json:"HostName"`
	DNSName      string   `json:"DNSName"`
	OS           string   `json:"OS"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Online       bool     `json:"Online"`
	Active       bool     `json:"Active"`
	LastSeen     string   `json:"LastSeen"`
}

func ReadTailscaleDevices(ctx context.Context) TailscaleDeviceStatus {
	status := TailscaleDeviceStatus{
		Devices:   []TailscaleDevice{},
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
	output, err := exec.CommandContext(ctx, "tailscale", "status", "--json").Output()
	if err != nil {
		status.Error = err.Error()
		return status
	}
	var raw tailscaleStatusJSON
	if err := json.Unmarshal(output, &raw); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Tailnet = raw.CurrentTailnet.Name
	if raw.Self.ID != "" || raw.Self.HostName != "" {
		status.Devices = append(status.Devices, tailscaleDeviceFromNode(raw.Self, true, raw.BackendState == "Running"))
	}
	for _, peer := range raw.Peer {
		status.Devices = append(status.Devices, tailscaleDeviceFromNode(peer, false, false))
	}
	sort.SliceStable(status.Devices, func(i, j int) bool {
		if status.Devices[i].Online != status.Devices[j].Online {
			return status.Devices[i].Online
		}
		if status.Devices[i].Self != status.Devices[j].Self {
			return status.Devices[i].Self
		}
		return strings.ToLower(status.Devices[i].Name) < strings.ToLower(status.Devices[j].Name)
	})
	return status
}

func tailscaleDeviceFromNode(node tailscaleNode, self bool, selfOnline bool) TailscaleDevice {
	ip := firstIPv4(node.TailscaleIPs)
	if ip == "" && len(node.TailscaleIPs) > 0 {
		ip = node.TailscaleIPs[0]
	}
	name := strings.TrimSpace(node.HostName)
	if name == "" {
		name = strings.TrimSuffix(node.DNSName, ".")
	}
	online := node.Online || selfOnline
	return TailscaleDevice{
		ID:          node.ID,
		Name:        name,
		DNSName:     strings.TrimSuffix(node.DNSName, "."),
		OS:          node.OS,
		IP:          ip,
		Online:      online,
		Active:      node.Active,
		Self:        self,
		LastSeen:    normalizeZeroTime(node.LastSeen),
		OpenClawURL: openClawURLForTailscaleIP(ip),
	}
}

func firstIPv4(ips []string) string {
	for _, ip := range ips {
		if !strings.Contains(ip, ":") {
			return ip
		}
	}
	return ""
}

func openClawURLForTailscaleIP(ip string) string {
	if strings.TrimSpace(ip) == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:18800", ip)
}

func normalizeZeroTime(value string) string {
	if strings.HasPrefix(value, "0001-01-01") {
		return ""
	}
	return value
}
