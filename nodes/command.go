package nodes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/exp/slog"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*Command)(nil)

// Command executes command lines.
type Command struct {
	// Description is used as the nodes identifier.
	Description string

	// The binary to run.
	// Defaults to 'bash'.
	Binary string

	// Arguments to pass when invoking the binary.
	// Each argument is templated.
	// Defaults to ["{{.Custom.script}}"] if .Script is not empty, [] otherwise.
	Arguments []string

	// A script to template and pass to .Binary.
	Script string

	// Input is passed to the process as stdin line by line.
	// Each line is templated.
	//
	// Useful when executing a binary that expects input (e.g. a password for authenticating).
	Input []string
}

func (sh Command) Shape() tensile.Shape {
	return tensile.Noop
}

func (sh Command) Identifier() string {
	return sh.Description
}

func (sh *Command) Validate() error {
	if sh.Binary == "" {
		sh.Binary = "bash"
	}

	if len(sh.Arguments) == 0 && len(sh.Script) > 0 {
		sh.Arguments = []string{
			"{{.Custom.script}}",
		}
	}

	return nil
}

type CommandOutput struct {
	Result         int
	Stdout, Stderr string
}

func (sh Command) Execute(ctx tensile.Context) (any, error) {
	templateCustom := map[string]any{}

	// template script
	if len(sh.Script) > 0 {
		f, err := os.CreateTemp("", "tensile-*")
		if err != nil {
			return nil, fmt.Errorf("nodes: error creating temporary file: %w", err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if err := tensile.Template(ctx.Facts(), sh.Script, f, templateCustom); err != nil {
			return nil, fmt.Errorf("nodes: error templating script to temporary file: %w", err)
		}

		templateCustom["script"] = f.Name()
	}

	// create exec.Command
	binPath, err := exec.LookPath(sh.Binary)
	if err != nil {
		return nil, fmt.Errorf("nodes: unable to find path for %q: %w", sh.Binary, err)
	}

	arguments := make([]string, len(sh.Arguments))
	for i, arg := range sh.Arguments {
		s, err := tensile.TemplateString(ctx.Facts(), arg, templateCustom)
		if err != nil {
			return nil, fmt.Errorf("nodes: error templating argument %d %q: %w", i, arg, err)
		}
		arguments[i] = s
	}

	ctx.Logger().Debug("command line",
		slog.String("binary", sh.Binary),
		slog.String("binpath", binPath),
		slog.Any("raw arguments", sh.Arguments),
		slog.Any("arguments", arguments),
	)
	cmd := exec.CommandContext(ctx.Context(), binPath, arguments...)

	wg := &sync.WaitGroup{}

	if len(sh.Input) > 0 {
		wg.Add(1)
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("nodes: error getting stdin pipe: %w", err)
		}

		go func() {
			defer stdinPipe.Close()
			defer wg.Done()
			for _, line := range sh.Input {
				if _, err := stdinPipe.Write([]byte(line + "\n")); err != nil {
					// TODO
				}
			}
		}()
	}

	// TODO support combined output as well

	stdout := &strings.Builder{}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("nodes: error getting stdout pipe: %w", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(stdout, stdoutPipe); err != nil {
			// TODO
		}
		ctx.Logger().Debug("stdout", slog.String("lines", stdout.String()))
	}()

	stderr := &strings.Builder{}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("nodes: error getting stderr pipe: %w", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(stderr, stderrPipe); err != nil {
			// TODO
		}
		ctx.Logger().Debug("stderr", slog.String("lines", stderr.String()))
	}()

	makeOutput := func(msg string, err error) (*CommandOutput, error) {
		wg.Wait()

		out := &CommandOutput{
			Result: cmd.ProcessState.ExitCode(),
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}

		if err == nil {
			return out, nil
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			out.Result = exitErr.ExitCode()
			if out.Result < 0 || out.Result >= 126 {
				// exit codes lower than zero are kernel errors
				// exit codes higher/equal 126 are generic errors
				return out, fmt.Errorf("nodes: %s: %w", msg, exitErr)
			}
		}

		// if the error is not exec.ExitError a larger issue is likely present
		return out, fmt.Errorf("nodes: %s: %w", msg, err)
	}

	// execute
	if err := cmd.Start(); err != nil {
		return makeOutput("error starting process", err)
	}

	if err := cmd.Wait(); err != nil {
		return makeOutput("error waiting on process", err)
	}

	return makeOutput("process finished", err)
}
