package cmd

// UserProjectWithHomeCmd inits a use command instance with home
func UseProjectWithHomeCmd(project, home string) *XsCommand {
	command := new(XsCommand)
	command.Cmd = XS_CMD_USE
	command.Buf = project
	if len(home) > 0 {
		command.Buf1 = home
	}
	return command
}

// UseProjectCmd inits a use command instance with out home
func UseProjectCmd(project string) *XsCommand {
	return UseProjectWithHomeCmd(project, "")
}
