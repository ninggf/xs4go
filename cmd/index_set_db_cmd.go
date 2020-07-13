package cmd

// IndexSetDbCmd
func IndexSetDbCmd(db string) *XsCommand {
	command := new(XsCommand)
	command.Cmd = XS_CMD_INDEX_SET_DB
	command.Buf = db
	return command
}
