package dsl

func (p *Parser) initRegistry() {
	p.cmdlookup = map[string]map[string]interface{}{
		"server": {
			"listdb": p.parseListDatabasesCmd,
			"newdb":  p.parseNewDatabaseCmd,
			"init":   p.parseServerInitCmd,
			"dropdb": p.parseDropDatabaseCmd,
			"login":  p.parseLoginCmd,
		},
		"user": {
			"new":    p.parseNewUserCmd,
			"all":    p.parseListUsersCmd,
			"about":  p.parseUserInfoCmd,
			"delete": p.parseDropUserCmd,
			"passw":  p.parseNewPasswordCmd,
			"access": p.parseUserSystemAccessCmd,
			"db":     p.parseUserDatabaseAccessCmd,
			"whoami": p.parseWhoamiCmd,
		},
		"database": {
			"newdir":      p.parseNewDirectoryCmd,
			"newfile":     p.parseNewFileCmd,
			"listdir":     p.parseListDirectoryCmd,
			"rename":      p.parseRenameContentCmd,
			"move":        p.parseMoveContentCmd,
			"copy":        p.parseCopyContentCmd,
			"delete":      p.parseDeleteContentCmd,
			"info":        p.parseContentInfoCmd,
			"makepublic":  p.parseMakeContentPublicCmd,
			"makeprivate": p.parseMakeContentPrivateCmd,
			"readfile":    p.parseReadFileCmd,
			"modfile":     p.parseModifyFileCmd,
			"deletebytes": p.parseDeleteAttachmentCmd,
			"counter":     p.parseCounterCmd,
			"select":      p.parseSelectCmd,
			"set":         p.parseSetCmd,
			"unset":       p.parseUnsetCmd,
		},
	}
}
