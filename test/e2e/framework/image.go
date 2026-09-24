package framework

import "io"

// Image describes how to run a target OS as a container machine.
type Image struct {
	// Ref is the image to start the container with.
	Ref string
	// Entrypoint is the entrypoint to start the container with.
	Entrypoint []string
	// CapAdd lists added Linux capabilities.
	CapAdd []string
	// SecurityOpt lists container security options.
	SecurityOpt []string
	// Tmpfs maps mount points to mount options.
	Tmpfs map[string]string
	// Files maps container paths to file content copied in before start.
	// Wrap strings in [strings.NewReader].
	Files map[string]io.Reader
	// WaitCmd probes readiness inside the container.
	// Exit codes 0 and 1 count as ready so a degraded systemd boot, common in containers, does not fail the wait.
	WaitCmd []string
}

// SystemdDebian is a Debian 12 machine with systemd as PID 1.
var SystemdDebian = Image{
	Ref: "quay.io/k0sproject/bootloose-debian12:v0.9.6",
	// Some docker setups mount cgroup2 read-only even with a private
	// cgroup namespace, others mount it read-write but refuse a
	// remount, so remount only when it is not writable.
	Entrypoint: []string{
		"/bin/sh", "-c",
		"[ -w /sys/fs/cgroup ] || mount -o remount,rw /sys/fs/cgroup && exec /sbin/init",
	},
	CapAdd: []string{"SYS_ADMIN"},
	// docker's default AppArmor profile denies mount(2) even with
	// CAP_SYS_ADMIN, breaking the cgroup remount on AppArmor hosts
	SecurityOpt: []string{"apparmor=unconfined"},
	Tmpfs: map[string]string{
		"/run":      "",
		"/run/lock": "",
		"/tmp":      "",
	},
	WaitCmd: []string{"systemctl", "is-system-running", "--wait"},
}

// OpenWrtImages are the OpenWrt releases every openwrt* test runs on.
var OpenWrtImages = map[string]Image{
	"24.10": OpenWrt2410,
	"25.12": OpenWrt2512,
}

// OpenWrt2410 is an OpenWrt 24.10 machine with procd as PID 1 and opkg.
var OpenWrt2410 = Image{
	Ref:        "docker.io/openwrt/rootfs:x86_64-24.10.8",
	Entrypoint: []string{"/sbin/init"},
	// procd wraps services in ujail, which needs to clone mount
	// namespaces: blocked by the default seccomp profile without
	// CAP_SYS_ADMIN and by docker's AppArmor profile on AppArmor hosts
	CapAdd:      []string{"SYS_ADMIN"},
	SecurityOpt: []string{"seccomp=unconfined", "apparmor=unconfined"},
	WaitCmd:     []string{"ubus", "call", "system", "board"},
}

// OpenWrt2512 is an OpenWrt 25.12 machine with procd as PID 1 and apk.
var OpenWrt2512 = Image{
	Ref:        "docker.io/openwrt/rootfs:x86_64-25.12.4",
	Entrypoint: []string{"/sbin/init"},
	// see OpenWrt2410
	CapAdd:      []string{"SYS_ADMIN"},
	SecurityOpt: []string{"seccomp=unconfined", "apparmor=unconfined"},
	WaitCmd:     []string{"ubus", "call", "system", "board"},
}

// sleepEntrypoint keeps a plain container running without an init.
var sleepEntrypoint = []string{"/bin/sh", "-c", "sleep infinity"}

// ArchLinux is a plain Arch container.
var ArchLinux = Image{
	Ref:        "docker.io/library/archlinux:base",
	Entrypoint: sleepEntrypoint,
	WaitCmd:    []string{"pacman", "--version"},
}

// Alpine is a plain Alpine container.
var Alpine = Image{
	Ref:        "docker.io/library/alpine:3.22",
	Entrypoint: sleepEntrypoint,
	WaitCmd:    []string{"apk", "--version"},
}
