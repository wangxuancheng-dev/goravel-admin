package bootstrap

import (
	"slices"
	"testing"

	contractsconsole "github.com/goravel/framework/contracts/console"
	frameworkconsole "github.com/goravel/framework/console"

	"goravel/app/console"
)

func signatures(cmds []contractsconsole.Command) []string {
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, c.Signature())
	}
	return out
}

// Every app command (including all scheduled ones) must survive the production filter.
func TestProductionCommandsAllowlistKeepsAllAppCommands(t *testing.T) {
	cmds := (&console.Kernel{}).Commands()
	kept := signatures(frameworkconsole.FilterCommandsByAllowlist(cmds, ProductionCommandsAllowlist(false)))
	for _, sig := range signatures(cmds) {
		if !slices.Contains(kept, sig) {
			t.Errorf("command %q is dropped in production; add its prefix to ProductionCommandsAllowlist", sig)
		}
	}
}

type sigOnly struct {
	contractsconsole.Command
	sig string
}

func (s sigOnly) Signature() string { return s.sig }

func TestProductionCommandsAllowlistMaintenanceToggle(t *testing.T) {
	cmds := []contractsconsole.Command{sigOnly{sig: "up"}, sigOnly{sig: "down"}, sigOnly{sig: "migrate"}, sigOnly{sig: "schedule:run"}, sigOnly{sig: "make:model"}}

	off := signatures(frameworkconsole.FilterCommandsByAllowlist(cmds, ProductionCommandsAllowlist(false)))
	if !slices.Equal(off, []string{"migrate", "schedule:run"}) {
		t.Fatalf("maintenance off: got %v", off)
	}

	on := signatures(frameworkconsole.FilterCommandsByAllowlist(cmds, ProductionCommandsAllowlist(true)))
	if !slices.Equal(on, []string{"up", "down", "migrate", "schedule:run"}) {
		t.Fatalf("maintenance on: got %v", on)
	}
}
