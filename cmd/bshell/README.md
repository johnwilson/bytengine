## Bytengine CLI

Shell for **[Bytengine](https://github.com/johnwilson/bytengine "Bytengine")**

## Installation

Bshell has only been tested on Mac OS X and Linux

Build Prerequisites:

* **[Go](http://golang.org/doc/install "Go")**

Assuming you have set up $GOPATH properly and added $GOPATH/bin to $PATH do the
following:

1. Get bshell `go get github.com/johnwilson/bytengine/cmd/bshell`

2. Running bshell `bshell run -h`

## Quick Guide

Bshell enables you to run **[Bytengine](https://github.com/johnwilson/bytengine/ "Bytengine")** 
BQL (Bytengine Query Language) scripts and work with the resultset using javascript.
Let's run through a trivial example to give you and idea how it works.

This guide asumes you have a Bytengine instance running on *http://localhost:8500*
and your username is 'admin' with password 'password'.

### Connect to Bytengine
```
    bshell run -u=admin -p=password 
```

You should now have a ` bql> ` prompt where you can start issuing commands/queries.
Used **Ctrl+d** to exit bshell.

Bshell executes single line commands by default so pressing *enter* will execute
the script.

### Create a test database
```
    bql> server.newdb "test"
    {
        "data": true,
        "status": "ok"
    }
```

If you want to write a longer bql script you can enter the **\e** 
command to open the bql editor (which uses vim by default) to write your script 
whcich is executed after saving it.
```
    bql> \e
```

Bshell stores the last successful command resultset, the value of which can be 
recalled using the **'lastresult()'** javascript function. To run a javascript 
statement use **'\s'** before your statement.

### Get last result and return status in javascript
```
    bql> \s lastresult().status
    ok
```

As mentioned previously, if you need to execute a longer javascript script enter
**\es** command to open the javascript editor.
```
    bql> \es
```

### bshell commands

| Command | Description                               |
|---------|-------------------------------------------|
| \e      | Open external bql editor (vim by default) |
| \es     | Open external javascript editor           |
| \s      | Run javascript statement                  |
| \q      | Quit bshell (you can also use ctrl+d)     |

### javascript inbuilt functions for bytengine

* `lastresult()` Returns the last resultset from a bql query/command
* `writebytes(db, remotefile, localfile)` Writes a 'local file' to a 'bytengine file' byte layer
* `readbytes(db, remotefile, localfile)` Downloads a 'bytengine file' byte layer

## Some Handy Links

[Otto](https://github.com/robertkrimen/otto) - A JavaScript interpreter in Go

[liner](https://github.com/peterh/liner) - Pure Go line editor with history,
inspired by linenoise