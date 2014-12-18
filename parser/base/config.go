package base

import (
	"github.com/johnwilson/bytengine"
)

func init() {
	p := NewParser()

	// register server functions
	p.registry.NewServerItem("listdb", "dbs", p.parseListDatabasesCmd)
	p.registry.NewServerItem("newdb", "", p.parseNewDatabaseCmd)
	p.registry.NewServerItem("init", "", p.parseServerInitCmd)
	p.registry.NewServerItem("dropdb", "", p.parseDropDatabaseCmd)

	// register user functions
	p.registry.NewUserItem("new", "", p.parseNewUserCmd)
	p.registry.NewUserItem("all", "", p.parseListUsersCmd)
	p.registry.NewUserItem("about", "", p.parseUserInfoCmd)
	p.registry.NewUserItem("delete", "rm", p.parseDropUserCmd)
	p.registry.NewUserItem("passw", "", p.parseNewPasswordCmd)
	p.registry.NewUserItem("access", "", p.parseUserSystemAccessCmd)
	p.registry.NewUserItem("db", "", p.parseUserDatabaseAccessCmd)
	p.registry.NewUserItem("whoami", "", p.parseWhoamiCmd)

	// register database functions
	p.registry.NewDatabaseItem("newdir", "mkdir", p.parseNewDirectoryCmd)
	p.registry.NewDatabaseItem("newfile", "write", p.parseNewFileCmd)
	p.registry.NewDatabaseItem("listdir", "ls", p.parseListDirectoryCmd)
	p.registry.NewDatabaseItem("rename", "", p.parseRenameContentCmd)
	p.registry.NewDatabaseItem("move", "mv", p.parseMoveContentCmd)
	p.registry.NewDatabaseItem("copy", "cp", p.parseCopyContentCmd)
	p.registry.NewDatabaseItem("delete", "rm", p.parseDeleteContentCmd)
	p.registry.NewDatabaseItem("info", "", p.parseContentInfoCmd)
	p.registry.NewDatabaseItem("makepublic", "public", p.parseMakeContentPublicCmd)
	p.registry.NewDatabaseItem("makeprivate", "private", p.parseMakeContentPrivateCmd)
	p.registry.NewDatabaseItem("readfile", "read", p.parseReadFileCmd)
	p.registry.NewDatabaseItem("updatefile", "update", p.parseModifyFileCmd)
	p.registry.NewDatabaseItem("deletebytes", "", p.parseDeleteAttachmentCmd)
	p.registry.NewDatabaseItem("counter", "", p.parseCounterCmd)
	p.registry.NewDatabaseItem("select", "", p.parseSelectCmd)
	p.registry.NewDatabaseItem("set", "", p.parseSetCmd)
	p.registry.NewDatabaseItem("unset", "", p.parseUnsetCmd)

	bytengine.RegisterParser("base", p)
}
