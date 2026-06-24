package webapp

import "testing"

func TestTailscaleDeviceFromNodeBuildsOpenClawURL(t *testing.T) {
	device := tailscaleDeviceFromNode(tailscaleNode{
		ID:           "node-1",
		HostName:     "donald",
		TailscaleIPs: []string{"100.78.187.25", "fd7a:115c:a1e0::1"},
		Online:       true,
	}, false, false)

	if !device.Online {
		t.Fatal("device should be online")
	}
	if device.IP != "100.78.187.25" {
		t.Fatalf("ip = %q, want 100.78.187.25", device.IP)
	}
	if device.OpenClawURL != "http://100.78.187.25:18800" {
		t.Fatalf("openclaw url = %q", device.OpenClawURL)
	}
}

func TestTailscaleSelfUsesBackendRunningAsOnline(t *testing.T) {
	device := tailscaleDeviceFromNode(tailscaleNode{
		ID:           "self",
		HostName:     "local",
		TailscaleIPs: []string{"100.85.64.66"},
	}, true, true)

	if !device.Self {
		t.Fatal("device should be marked self")
	}
	if !device.Online {
		t.Fatal("self should be online when backend is running")
	}
}
