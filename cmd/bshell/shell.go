package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/client"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)

type stateFn func(*Shell) stateFn

// meta-commands constants
const (
	OpenBQLEditor        string = "\\e"
	OpenJavascriptEditor string = "\\es"
	RunJavascript        string = "\\s"
	Quit                 string = "\\q"
)

// BShell
type Shell struct {
	historyFile   string
	editorFileBQL string
	editorFileJS  string
	BEClient      *client.Client
	state         stateFn
	line          *liner.State
	jsengine      *otto.Otto // javascript engine
	input         string
	lastresult    string // last bql result
	editorName    string
}

// Write output to terminal
func (sh *Shell) write(msg string) {
	fmt.Println(msg)
}

// Write error to terminal
func (sh *Shell) writeError(e error) {
	fmt.Printf("error: %s\n", e)
}

// Launch external editor such as nano, vim or subl
func (sh *Shell) launchEditor(fpath string) (bool, error) {
	// get file last modified date/time
	fi, err := os.Stat(fpath)
	if err != nil {
		return false, err
	}
	dt1 := fi.ModTime() // date/time before editing

	cmd := exec.Command(sh.editorName, fpath)

	// redirect input/output/error
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start editor and wait untile closed
	err = cmd.Start()
	if err != nil {
		return false, err
	}
	err = cmd.Wait()
	if err != nil {
		return false, err
	}

	// get file last modified date/time
	fi, err = os.Stat(fpath)
	if err != nil {
		return false, err
	}
	dt2 := fi.ModTime() // date/time before editing

	// check to see if file has been edited
	if !dt1.Before(dt2) {
		return false, nil
	}

	// load edited content into shell.input
	f, err := os.Open(fpath)
	defer f.Close()
	if err != nil {
		return false, err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return false, err
	}

	// update input
	sh.input = string(b)

	return true, nil
}

// Execute BQL
func (sh *Shell) execBQL(editor bool) stateFn {
	out, err := sh.BEClient.Exec(sh.input, 0)
	if err != nil {
		out = bytengine.ErrorResponse(err).String()
	}

	sh.write(out)
	if !editor {
		sh.line.AppendHistory(sh.input)
	}
	sh.lastresult = out
	return bqlPrompt
}

// Execute Javascript
func (sh *Shell) execScript(editor bool) stateFn {
	var out string
	var value otto.Value
	var err error

	if !editor {
		tmp := strings.TrimPrefix(sh.input, RunJavascript) // remove script meta-command
		value, err = sh.jsengine.Run(tmp)
	} else {
		value, err = sh.jsengine.Run(sh.input)
	}

	if err != nil {
		out = err.Error()
	} else if value.IsDefined() {
		out = value.String()
	}

	sh.write(out)
	if !editor {
		sh.line.AppendHistory(sh.input)
	}

	return bqlPrompt
}

// Helper function to create file
func (sh *Shell) touchFile(f string) error {
	if _, e := os.Stat(f); e != nil {
		if e := ioutil.WriteFile(f, []byte{}, 0777); e != nil {
			return e
		}
	}
	return nil
}

// Handle shell errors
func (sh *Shell) error(err error) stateFn {
	// check for EOF error i.e. Ctrl+D
	if err.Error() == "EOF" {
		sh.write("\nbye")
	} else {
		sh.writeError(err)
	}

	return bqlQuit
}

// Quit shell stateFn
func bqlQuit(sh *Shell) stateFn {
	// return nil stateFn to break out of loop
	return nil
}

// Launch external bql editor stateFn
func bqlEditor(sh *Shell) stateFn {
	changed, err := sh.launchEditor(sh.editorFileBQL)
	if err != nil {
		return sh.error(err)
	}

	// only execute if file changed
	if changed {
		return sh.execBQL(true)
	}
	return bqlPrompt
}

// Launch external script editor stateFn
func scriptEditor(sh *Shell) stateFn {
	changed, err := sh.launchEditor(sh.editorFileJS)
	if err != nil {
		return sh.error(err)
	}

	// only execute if file changed
	if changed {
		return sh.execScript(true)
	}
	return bqlPrompt
}

// Shell Prompt stateFn
func bqlPrompt(sh *Shell) stateFn {
	//sh.input = "" // clear input
	input, e := sh.line.Prompt("bql> ")
	if e != nil {
		return sh.error(e)
	}
	input = strings.Trim(input, " ")
	switch {
	case input == Quit:
		sh.write("\nbye")
		return bqlQuit
	case input == OpenBQLEditor:
		return bqlEditor
	case input == OpenJavascriptEditor:
		return scriptEditor
	case strings.HasPrefix(input, RunJavascript) == true:
		sh.input = input
		return sh.execScript(false)
	default:
		sh.input = input
		return sh.execBQL(false)
	}
}

// Start shell loop
func (sh *Shell) Start() {
	for sh.state != nil {
		sh.state = sh.state(sh)
	}
}

// Close and clean up
func (sh *Shell) Close() {
	if f, e := os.Create(sh.historyFile); e != nil {
		sh.write(fmt.Sprintf("Error writing history file: ", e))
	} else {
		sh.line.WriteHistory(f)
		f.Close()
	}

	sh.line.Close()
	sh.line = nil
}

// Initialise the shell
func (sh *Shell) Init(editorName string) error {
	// create directory
	dir := path.Join(os.TempDir(), "bshell")
	if _, err := os.Stat(dir); err != nil {
		if err = os.Mkdir(dir, 0777); err != nil {
			return err
		}
	}

	// create files
	sh.editorFileBQL = path.Join(dir, "bql_editor.txt")
	if err := sh.touchFile(sh.editorFileBQL); err != nil {
		return err
	}
	sh.editorFileJS = path.Join(dir, "js_editor.js")
	if err := sh.touchFile(sh.editorFileJS); err != nil {
		return err
	}
	sh.historyFile = path.Join(dir, "history")
	if err := sh.touchFile(sh.historyFile); err != nil {
		return err
	}

	// create shell
	sh.line = liner.NewLiner()
	// setup history
	if f, err := os.Open(sh.historyFile); err == nil {
		sh.line.ReadHistory(f)
		f.Close()
	}
	// set editor
	sh.editorName = editorName

	return nil
}

// Create a new shell
func NewShell() *Shell {
	bclient := client.NewClient()
	sh := Shell{
		BEClient: bclient,
		state:    bqlPrompt,
		jsengine: otto.New(),
	}

	// set javascript function: lastresult
	sh.jsengine.Set("lastresult", func(call otto.FunctionCall) otto.Value {
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(sh.lastresult), &obj)
		if err != nil {
			return otto.NullValue()
		}

		val, err := sh.jsengine.ToValue(obj)
		if err != nil {
			return otto.NullValue()
		}
		return val
	})

	// set javascript function: writebytes
	sh.jsengine.Set("writebytes", func(call otto.FunctionCall) otto.Value {
		db, err := call.Argument(0).ToString()
		if err != nil {
			sh.write("error: database must be a string")
			return otto.NullValue()
		}
		remotefile, err := call.Argument(1).ToString()
		if err != nil {
			sh.write("error: remote file must be a string")
			return otto.NullValue()
		}
		localfile, err := call.Argument(2).ToString()
		if err != nil {
			sh.write("error: local file must be a string")
			return otto.NullValue()
		}

		err = sh.BEClient.WriteBytes(db, remotefile, localfile)
		if err != nil {
			sh.write("error: " + err.Error())
			return otto.NullValue()
		}

		return otto.TrueValue()
	})

	// set javascript function: readbytes
	sh.jsengine.Set("readbytes", func(call otto.FunctionCall) otto.Value {
		db, err := call.Argument(0).ToString()
		if err != nil {
			sh.write("error: database must be a string")
			return otto.NullValue()
		}
		remotefile, err := call.Argument(1).ToString()
		if err != nil {
			sh.write("error: remote file must be a string")
			return otto.NullValue()
		}
		localfile, err := call.Argument(2).ToString()
		if err != nil {
			sh.write("error: local file must be a string")
			return otto.NullValue()
		}

		err = sh.BEClient.ReadBytes(db, remotefile, localfile)
		if err != nil {
			sh.write("error: " + err.Error())
			return otto.NullValue()
		}

		return otto.TrueValue()
	})

	return &sh
}
