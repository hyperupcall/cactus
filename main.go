package main

import (
	"fmt"
	"os"
	"strings"

	g "github.com/AllenDang/giu"
	"github.com/AllenDang/giu/imgui"
	"github.com/eankeen/cactus/cfg"
	cmd "github.com/eankeen/cactus/cmd"
	"github.com/eankeen/cactus/keymap"
	"github.com/eankeen/cactus/util"
	"github.com/fsnotify/fsnotify"
	cli "github.com/urfave/cli/v2"
)

// The key the user wants to run. By default, it's blank
var myCmd = cmd.New()

func loop(cfg *cfg.Cfg, keybinds *cfg.Keybinds) {
	// Quit on Escape
	if g.IsKeyDown(g.KeyEscape) {
		os.Exit(0)
	}

	// 'key' is each key in the config file,
	// The properties of each key is enumerated in the
	// members of keybindEntry
	for key, keybindEntry := range *keybinds {
		// If there is a hypthen in a config key, assume
		// it contains a modifier
		if strings.Contains(key, "-") {
			strs := strings.Split(key, "-")
			mod := strs[0]
			key = strs[1]

			if mod == "Shift" && (g.IsKeyDown(g.KeyLeftShift) || g.IsKeyDown(g.KeyRightShift)) {
				myCmd.RunCmdOnce(mod, key, keybindEntry)
				break
			} else if mod == "Control" && (g.IsKeyDown(g.KeyLeftControl) || g.IsKeyDown(g.KeyRightControl)) {
				myCmd.RunCmdOnce(mod, key, keybindEntry)
				break
			} else if mod == "Alt" && (g.IsKeyDown(g.KeyLeftAlt) || g.IsKeyDown(g.KeyRightAlt)) {
				myCmd.RunCmdOnce(mod, key, keybindEntry)
				break
			}
		} else {
			if g.IsKeyDown(keymap.Keymap[key]) {
				myCmd.RunCmdOnce("", key, keybindEntry)
				break
			}
		}
	}

	if myCmd.HasRan {
		// Exit if there is a success and we don't want to show info on success
		fmt.Println(myCmd.Result.Err, !myCmd.Keybind.InfoOnSuccess)
		if myCmd.Result.Err == nil && !myCmd.Keybind.InfoOnSuccess {
			os.Exit(0)
		}
	}

	var widgets []g.Widget

	// If we actually ran a command, and there is an error,
	// show the error instead of the hotkey table
	if myCmd.HasRan {
		// RESULT
		if myCmd.Result.Err != nil {
			widgets = append(widgets, g.Line(
				g.ArrowButton("Arrow", g.DirectionRight),
				g.Label("RESULT"),
			))

			widgets = append(widgets, g.Label("Error: "+myCmd.Result.Err.Error()))
			widgets = append(widgets, g.Label("ExecName: "+myCmd.Result.ExecName))
			widgets = append(widgets, g.Label("ExecArgs: ["))
			for _, arg := range myCmd.Result.ExecArgs {
				widgets = append(widgets, g.Label(
					fmt.Sprintf("  '%s'", arg),
				))
			}
			widgets = append(widgets, g.Label("]"))
			widgets = append(widgets, g.Label(""))
		}

		// SYSTEMD-RUN
		systemdRunOutput := util.ParseSystemdRunOutput(myCmd.Result.Output)
		widgets = append(widgets, g.Line(
			g.ArrowButton("Arrow", g.DirectionRight),
			g.Label("SYSTEMD-RUN"),
		))

		for _, keyValue := range systemdRunOutput {
			key := keyValue[0]
			value := keyValue[1]

			widgets = append(widgets, g.Line(
				g.Button(key).OnClick(func() {
					util.CopyToClipboard(value)
				}),
				g.Label(value),
			))
		}
		widgets = append(widgets, g.Label(""))

		// KEY
		widgets = append(widgets, g.Line(
			g.ArrowButton("Arrow", g.DirectionRight),
			g.Label("KEY"),
		))

		widgets = append(widgets, g.Line(
			g.Button("As").OnClick(func() {
				util.CopyToClipboard(myCmd.Keybind.As)
			}),
			g.Label(myCmd.Keybind.As),
		))
		widgets = append(widgets, g.Line(
			g.Button("Cmd").OnClick(func() {
				util.CopyToClipboard(myCmd.Keybind.Cmd)
			}),
			g.Label(myCmd.Keybind.Cmd),
		))
		widgets = append(widgets, g.Line(
			g.Button("Args").OnClick(func() {
				util.CopyToClipboard(strings.Join(myCmd.Keybind.Args, " "))
			}),
			g.Label("["),
		))
		for _, arg := range myCmd.Keybind.Args {
			widgets = append(widgets, g.Label(
				fmt.Sprintf("  '%s'", arg),
			))
		}

		widgets = append(widgets, g.Label("]"))

		widgets = append(widgets, g.Line(
			g.Button("Wait").OnClick(func() {
				util.CopyToClipboard(fmt.Sprintf("%t", myCmd.Keybind.Wait))
			}),
			g.Label(fmt.Sprintf("%t", myCmd.Keybind.Wait)),
		))
		widgets = append(widgets, g.Line(
			g.Button("Key").OnClick(func() {
				util.CopyToClipboard(myCmd.KeybindKey)
			}),
			g.Label(myCmd.KeybindKey),
		))
		widgets = append(widgets, g.Line(
			g.Button("Mod").OnClick(func() {
				util.CopyToClipboard(myCmd.KeybindMod)
			}),
			g.Label(myCmd.KeybindMod),
		))
		widgets = append(widgets, g.Label(""))

		// RAW OUTPUT
		if myCmd.Result.Output != "" {
			widgets = append(widgets, g.Line(
				g.ArrowButton("Arrow", g.DirectionRight),
				g.Label("RAW OUTPUT"),
			))
			widgets = append(widgets, g.Button("Copy Raw Output").OnClick(func() {
				util.CopyToClipboard(myCmd.Result.Output)
			}))
			widgets = append(widgets, g.Label(myCmd.Result.Output))
		}
	} else {
		table := g.Table("Command Table").FastMode(true).Rows(util.BuildGuiTableRows(*keybinds)...).Flags(
			imgui.TableFlags_Resizable | imgui.TableFlags_RowBg | imgui.TableFlags_Borders | imgui.TableFlags_SizingFixedFit | imgui.TableFlags_ScrollX | imgui.TableFlags_ScrollY | imgui.TableFlags_ScrollY,
		)
		widgets = append(widgets, table)
	}

	g.SingleWindow("Runner").Layout(
		widgets...,
	)
}

func main() {
	app := &cli.App{
		Name:    "cactus",
		Usage:   "Small hotkey application",
		Version: "0.3.0",
		Authors: []*cli.Author{
			{
				Name:  "Edwin Kofler",
				Email: "edwin@kofler.dev",
			},
		},
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "binds",
				Aliases: []string{"b"},
				Value:   util.GetCfgFile("binds.toml"),
				Usage:   "Location of bindings",
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   util.GetCfgFile("cactus.toml"),
				Usage:   "Location of configuration file",
			},
		},
		Action: func(c *cli.Context) error {
			// Configuration
			keybindsMnger := cfg.NewKeybindsMnger(c.String("binds"))
			err := keybindsMnger.Reload()
			util.Handle(err)

			cfgMnger := cfg.NewCfgMnger(c.String("config"))
			err = cfgMnger.Reload()
			util.Handle(err)

			cfg := cfgMnger.Get()
			keybinds := keybindsMnger.Get()

			// Watcher
			watcher, err := fsnotify.NewWatcher()
			util.Handle(err)
			defer watcher.Close()

			done := make(chan bool)
			go func() {
				for {
					select {
					case event, ok := <-watcher.Events:
						if !ok {
							return
						}

						if event.Op&fsnotify.Write == fsnotify.Write {
							err = keybindsMnger.Reload()
							util.Handle(err)

							err = cfgMnger.Reload()
							util.Handle(err)

							// TODO: Manually rerender?
							// imgui.NewFrame()
							// imgui.Render()
							// drawData := imgui.RenderedDrawData()
							// fmt.Println(drawData.Valid())
						}
					case err, ok := <-watcher.Errors:
						if !ok {
							return
						}

						util.Handle(err)
					}
				}
			}()

			err = watcher.Add(c.String("binds"))
			util.Handle(err)
			err = watcher.Add(c.String("config"))
			util.Handle(err)

			// Imgui
			ctx := imgui.CreateContext(nil)
			err = ctx.SetCurrent()
			util.Handle(err)

			if cfg.FontFile != "" {
				imgui.CurrentIO().Fonts().AddFontFromFileTTF(cfg.FontFile, float32(cfg.FontSize))
			}

			wnd := g.NewMasterWindow("Cactus", 800, 450, g.MasterWindowFlagsNotResizable|g.MasterWindowFlagsFloating, nil)

			wnd.Run(func() {
				// fmt.Println(keybinds)
				// b := true
				// imgui.ShowDemoWindow(&b)
				loop(cfg, keybinds)
			})

			<-done
			return nil
		},
	}

	err := app.Run(os.Args)
	util.Handle(err)
}
