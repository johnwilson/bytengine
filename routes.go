package bytengine

func Initialize(r *Router) {
	// user management commands
	r.Add("user.new", userNew)
	r.Add("user.all", userAll)
	r.Add("user.about", userAbout)
	r.Add("user.delete", userDelete)
	r.Add("user.passw", userPassw)
	r.Add("user.access", userAccess)
	r.Add("user.db", userDb)
	r.Add("user.whoami", userWhoami)

	// server management commands
	r.Add("server.listdb", serverListDb)
	r.Add("server.newdb", serverNewDb)
	r.Add("server.init", serverInit)
	r.Add("server.dropdb", serverDropDb)

	// content management commands
	r.Add("database.newdir", dbNewDir)
	r.Add("database.newfile", dbNewFile)
	r.Add("database.listdir", dbListDir)
	r.Add("database.rename", dbRename)
	r.Add("database.move", dbMove)
	r.Add("database.copy", dbCopy)
	r.Add("database.delete", dbDelete)
	r.Add("database.info", dbInfo)
	r.Add("database.makepublic", dbMakePublic)
	r.Add("database.makeprivate", dbMakePrivate)
	r.Add("database.readfile", dbReadFile)
	r.Add("database.modfile", dbModFile)
	r.Add("database.deletebytes", dbDeleteBytes)
	r.Add("database.counter", dbCounter)
	r.Add("database.select", dbSelect)
	r.Add("database.set", dbSet)
	r.Add("database.unset", dbUnset)

	// internal commands
	r.Add("login", loginHandler)
	r.Add("uploadticket", uploadTicketHandler)
	r.Add("writebytes", writebytesHandler)
	r.Add("readbytes", readbytesHandler)
	r.Add("directaccess", direcaccessHandler)
}
