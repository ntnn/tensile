package std

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var _ ServiceManager = (*Procd)(nil)

func init() {
	RegisterServiceManager("procd", &Procd{})
}

// initDir holds procd init scripts.
const initDir = "/etc/init.d"

// Procd manages services through /etc/init.d scripts on OpenWrt.
type Procd struct {
	// run executes /etc/init.d/<name> <verb> and returns the exit code and combined output.
	// A non-nil error means the script could not be run at all.
	run func(ctx context.Context, name, verb string) (int, []byte, error)
	// available reports whether procd is present.
	available func() bool
	// dir overrides the init script directory.
	dir string
}

// Handles implements [ServiceManager].
func (p *Procd) Handles(_ context.Context, name string) (bool, error) {
	if !p.isAvailable() {
		return false, nil
	}

	info, err := os.Stat(p.initDir() + "/" + name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat init script %q: %w", name, err)
	}
	return !info.IsDir() && info.Mode()&0o100 != 0, nil
}

// Status implements [ServiceManager].
func (p *Procd) Status(ctx context.Context, name string) (ServiceStatus, error) {
	var status ServiceStatus

	code, out, err := p.initd(ctx, name, "enabled")
	if err != nil {
		return status, fmt.Errorf("init script %q enabled: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	status.Enabled = code == 0

	code, out, err = p.initd(ctx, name, "running")
	if err != nil {
		return status, fmt.Errorf("init script %q running: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	status.Active = code == 0

	return status, nil
}

// Apply implements [ServiceManager].
func (p *Procd) Apply(ctx context.Context, name string, desired ServiceStatus) error {
	current, err := p.Status(ctx, name)
	if err != nil {
		return err
	}

	if current.Enabled != desired.Enabled {
		verb := "enable"
		if !desired.Enabled {
			verb = "disable"
		}
		if err := p.transition(ctx, name, verb); err != nil {
			return err
		}
	}

	if current.Active != desired.Active {
		verb := "start"
		if !desired.Active {
			verb = "stop"
		}
		if err := p.transition(ctx, name, verb); err != nil {
			return err
		}
	}

	return nil
}

// Restart implements [ServiceManager].
func (p *Procd) Restart(ctx context.Context, name string) error {
	return p.transition(ctx, name, "restart")
}

func (p *Procd) transition(ctx context.Context, name, verb string) error {
	code, out, err := p.initd(ctx, name, verb)
	if err != nil {
		return fmt.Errorf("init script %q %s: %w: %s", name, verb, err, strings.TrimSpace(string(out)))
	}
	if code != 0 {
		return fmt.Errorf("init script %q %s: exit code %d: %s", name, verb, code, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *Procd) initd(ctx context.Context, name, verb string) (int, []byte, error) {
	if p.run != nil {
		return p.run(ctx, name, verb)
	}
	//nolint:gosec // path is fixed, name is the managed service
	out, err := exec.CommandContext(ctx, p.initDir()+"/"+name, verb).CombinedOutput()
	if err != nil {
		if exitedNonZero(err) {
			return exitCode(err), out, nil
		}
		return -1, out, err
	}
	return 0, out, nil
}

// exitCode extracts the exit code from an [exec.ExitError].
func exitCode(err error) int {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func (p *Procd) isAvailable() bool {
	if p.available != nil {
		return p.available()
	}
	_, err := os.Stat("/sbin/procd")
	return err == nil
}

func (p *Procd) initDir() string {
	if p.dir != "" {
		return p.dir
	}
	return initDir
}
