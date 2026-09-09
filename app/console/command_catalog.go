package console

// CommandDescriptionMap returns signature -> description for registered artisan commands.
func CommandDescriptionMap() map[string]string {
	kernel := Kernel{}
	commands := kernel.Commands()
	out := make(map[string]string, len(commands))
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		signature := cmd.Signature()
		if signature == "" {
			continue
		}
		out[signature] = cmd.Description()
	}
	return out
}
