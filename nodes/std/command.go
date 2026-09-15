package std

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Command)(nil)
var _ tensile.Validator = (*Command)(nil)
var _ tensile.Executor = (*Command)(nil)

// WellKnownShellArgs maps shells to arguments so that [Command.Command] can be executed by the shell.
// It is used to fill [Command.Args] if it is empty based on [Command.Shell].
var WellKnownShellArgs = map[string][]string{
	"dash":       {"-c"},
	"sh":         {"-c"},
	"bash":       {"-c"},
	"zsh":        {"-c"},
	"python":     {"-c"},
	"python3":    {"-c"},
	"ruby":       {"-e"},
	"perl":       {"-e"},
	"pwsh":       {"-NoProfile", "-Command"},
	"powershell": {"-NoProfile", "-Command"},
	"cmd":        {"/C"},
}

// Command runs a command line through an interpreter.
type Command struct {
	// Shell is the interpreter binary, e.g. "bash", "pwsh", "python".
	// Defaults to "sh", regardless of OS.
	//
	// This is intentional - defining the shell to depend on the OS is
	// error prone as POSIXly correct shells have vastly different
	// syntax than powershell. So in an environment with both Windows
	// and Unixoid machines would have to define both Shell and Command
	// depending on the OS anyway, unless e.g. sh/bash is also available
	// on Windows, in which case the Shell would have to be set for all
	// hosts anyway.
	// So to summarize - setting the shell per OS is nonsense and only
	// makes sense for homogeneous environments where you might as well
	// set the shell static if it diverges from the default.
	Shell string
	// Args are passed between interpreter and command.
	// Defaults are looked up by interpreter name.
	// Unknown interpreters require Args.
	Args []string
	// Command is passed as the final argument to the interpreter.
	Command string
	// Env is appended to the process environment as KEY=VALUE.
	Env map[string]string
	// Dir is the working directory. Empty means inherited.
	Dir string

	// Creates skips execution when the path exists.
	Creates string
	// Removes skips execution when the path is missing.
	Removes string

	// Unless skips execution when the command exits 0.
	// Executed with the same Shell, Args, Env and Dir as Command.
	Unless string
	// OnlyIf skips execution unless the command exits 0.
	// Executed with the same Shell, Args, Env and Dir as Command.
	OnlyIf string

	// run executes the interpreter and returns combined output.
	run func(wire tensile.Wire, args ...string) ([]byte, error)
}

// Identity implements [tensile.Identifier].
func (c *Command) Identity() tensile.Identity {
	return tensile.AsIdentity("command", "command", c.Command)
}

// Validate implements [tensile.Validator].
func (c *Command) Validate(_ tensile.Wire) error {
	if c.Command == "" {
		return errors.New("command is required")
	}

	shell := c.shell()
	if _, err := exec.LookPath(shell); err != nil {
		return fmt.Errorf("looking up interpreter %q: %w", shell, err)
	}

	if _, err := c.args(); err != nil {
		return err
	}
	return nil
}

// NeedsExecution implements [tensile.Executor].
func (c *Command) NeedsExecution(wire tensile.Wire) (bool, error) {
	guards := []func(tensile.Wire) (bool, error){
		c.createsGuard,
		c.removesGuard,
		c.unlessGuard,
		c.onlyIfGuard,
	}
	for _, guard := range guards {
		skip, err := guard(wire)
		if err != nil {
			return false, err
		}
		if skip {
			return false, nil
		}
	}
	return true, nil
}

// createsGuard skips execution when the Creates path exists.
func (c *Command) createsGuard(_ tensile.Wire) (bool, error) {
	if c.Creates == "" {
		return false, nil
	}
	_, err := os.Stat(c.Creates)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking creates path: %w", err)
}

// removesGuard skips execution when the Removes path is missing.
func (c *Command) removesGuard(_ tensile.Wire) (bool, error) {
	if c.Removes == "" {
		return false, nil
	}
	_, err := os.Stat(c.Removes)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, fmt.Errorf("checking removes path: %w", err)
}

// unlessGuard skips execution when Unless exits 0.
func (c *Command) unlessGuard(wire tensile.Wire) (bool, error) {
	if c.Unless == "" {
		return false, nil
	}
	_, err := c.exec(wire, c.Unless)
	return err == nil, nil
}

// onlyIfGuard skips execution unless OnlyIf exits 0.
func (c *Command) onlyIfGuard(wire tensile.Wire) (bool, error) {
	if c.OnlyIf == "" {
		return false, nil
	}
	_, err := c.exec(wire, c.OnlyIf)
	return err != nil, nil
}

// Execute implements [tensile.Executor].
func (c *Command) Execute(wire tensile.Wire) error {
	out, err := c.exec(wire, c.Command)
	if err != nil {
		return fmt.Errorf("running command: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if len(out) > 0 {
		wire.Logger().Debug("command output", "output", string(out))
	}
	return nil
}

func (c *Command) shell() string {
	if c.Shell == "" {
		return "sh"
	}
	return c.Shell
}

// args returns the interpreter arguments preceding the command.
func (c *Command) args() ([]string, error) {
	if c.Args != nil {
		return c.Args, nil
	}
	args, ok := WellKnownShellArgs[c.shell()]
	if !ok {
		return nil, fmt.Errorf("unknown interpreter %q: Args is required", c.shell())
	}
	return args, nil
}

// exec runs command through the interpreter and returns combined output.
func (c *Command) exec(wire tensile.Wire, command string) ([]byte, error) {
	args, err := c.args()
	if err != nil {
		return nil, err
	}
	args = append(args, command)

	if c.run != nil {
		return c.run(wire, args...)
	}

	//nolint:gosec // running user-supplied commands is the purpose of this node
	cmd := exec.CommandContext(wire.Context(), c.shell(), args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range c.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	return cmd.CombinedOutput()
}
