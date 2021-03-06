package run

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/google/uuid"
	"github.com/hyperupcall/cactus/cfg"
	"github.com/hyperupcall/cactus/util"
)

type Cmd struct {
	KeybindKey string
	KeybindMod string
	Keybind    cfg.KeybindEntry
	HasRan     bool
	Result     CmdResult
}

type CmdResult struct {
	ExecName string
	ExecArgs []string
	Err      error
	Output   string
}

func New() *Cmd {
	// This sets defaults
	return &Cmd{
		KeybindKey: "",
		KeybindMod: "",
		// Not defaults, overriden in RunCmdOnce
		Keybind: cfg.KeybindEntry{
			As:             "",
			Cmd:            "",
			Args:           []string{},
			Wait:           false,
			AlwaysShowInfo: false,
		},
		HasRan: false,
		Result: CmdResult{},
	}
}

func (cmd *Cmd) RunCmd() CmdResult {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return CmdResult{
			ExecName: "",
			ExecArgs: []string{},
			Err:      fmt.Errorf("Cactus Internal Error: Could not generate random number\n%w", err),
			Output:   "",
		}
	}

	args := []string{
		"--no-ask-password",
		"--unit", "cactus-" + uuid.String(),
		"--description", fmt.Sprintf("Cactus Start for command: '%s'", cmd.Keybind.Cmd),
		"--send-sighup",
		"--working-directory", os.Getenv("HOME"),
		"--user",
	}

	if cmd.Keybind.Wait {
		args = append(args, "--wait")
	}

	args = append(args, "--")

	// TODO custom ones
	switch cmd.Keybind.As {
	case "sh":
		_, err := os.Stat("/usr/bin/dash")
		if errors.Is(err, os.ErrNotExist) {
			args = append(args, "/usr/bin/sh", "-c", cmd.Keybind.Cmd)
		} else {
			// use dash directly if possible, even 'sh' is specified since
			// sometimes 'sh' is symlinked to bash
			args = append(args, "/usr/bin/dash", "-c", cmd.Keybind.Cmd)
		}
	case "bash":
		args = append(args, "/usr/bin/bash", "-c", cmd.Keybind.Cmd)
	default:
		cmd.Keybind.As = "exec"
		args = append(args, cmd.Keybind.Cmd)
		args = append(args, cmd.Keybind.Args...)
	}

	var templatedArgs = make([]string, len(args))
	for i, arg := range args {
		template := func(text string) string {
			tpl, err := template.New("foo").Funcs(sprig.TxtFuncMap()).Parse(arg)
			util.Handle(err)

			var b bytes.Buffer
			tpl.Execute(&b, struct{}{})

			return b.String()
		}

		templatedArgs[i] = template(arg)
	}

	execName := "/usr/bin/systemd-run"
	execCmd := exec.Command(execName, templatedArgs...)

	output, err := execCmd.CombinedOutput()
	return CmdResult{
		ExecName: execName,
		ExecArgs: templatedArgs,
		Err:      err,
		Output:   string(output),
	}
}
