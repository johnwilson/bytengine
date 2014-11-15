package engine

import (
	"github.com/johnwilson/bytengine/core"
)

func initialize(r *core.CommandRouter) {
	// user management commands
	r.AddCommandHandler("user.new", handler1)
	r.AddCommandHandler("user.all", handler2)
	r.AddCommandHandler("user.about", handler3)
	r.AddCommandHandler("user.delete", handler4)
	r.AddCommandHandler("user.passw", handler5)
	r.AddCommandHandler("user.access", handler6)
	r.AddCommandHandler("user.db", handler7)
	r.AddCommandHandler("user.whoami", handler8)

	// server management commands
	r.AddCommandHandler("server.listdb", handler9)
	r.AddCommandHandler("server.newdb", handler10)
	r.AddCommandHandler("server.init", handler11)
	r.AddCommandHandler("server.dropdb", handler12)

	// content management commands
	r.AddCommandHandler("database.newdir", handler13)
	r.AddCommandHandler("database.newfile", handler14)
	r.AddCommandHandler("database.listdir", handler15)
	r.AddCommandHandler("database.rename", handler16)
	r.AddCommandHandler("database.move", handler17)
	r.AddCommandHandler("database.copy", handler18)
	r.AddCommandHandler("database.delete", handler19)
	r.AddCommandHandler("database.info", handler20)
	r.AddCommandHandler("database.makepublic", handler21)
	r.AddCommandHandler("database.makeprivate", handler22)
	r.AddCommandHandler("database.readfile", handler23)
	r.AddCommandHandler("database.modfile", handler24)
	r.AddCommandHandler("database.deletebytes", handler25)
	r.AddCommandHandler("database.counter", handler26)
	r.AddCommandHandler("database.select", handler27)
	r.AddCommandHandler("database.set", handler28)
	r.AddCommandHandler("database.unset", handler29)

	// internal commands
	r.AddCommandHandler("login", loginHandler)
	r.AddCommandHandler("uploadticket", uploadTicketHandler)
	r.AddCommandHandler("writebytes", writebytesHandler)
	r.AddCommandHandler("readbytes", readbytesHandler)
	r.AddCommandHandler("directaccess", direcaccessHandler)
}
