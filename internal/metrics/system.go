package metrics

import "github.com/shirou/gopsutil/v3/host"

type System struct {
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
}

func collectSystem() (System, error) {
	info, err := host.Info()
	if err != nil {
		return System{}, err
	}

	arch := info.KernelArch
	switch arch {
	case "x86_64":
		arch = "x64"
	case "aarch64", "arm64":
		arch = "arm64"
	case "i386", "i686":
		arch = "x86"
	}

	return System{
		Hostname: info.Hostname,
		Platform: info.OS + " (" + arch + ")",
	}, nil
}
