package framework

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
