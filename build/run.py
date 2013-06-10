from docutils import core
from docutils.writers.html4css1 import Writer
from docutils.utils import SystemMessage
from jinja2 import FileSystemLoader, Environment
import os
import subprocess
import json
import argparse
import sys
import glob

import cmds

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)),"..")
NavBar = [
    {"name":"Home","url":"/"},
    {"name":"Documentation","url":"/docs"},
    {"name":"Terminal","url":"/terminal"},
    {"name":"Commands","url":"/commands"}
]

def setupDirs():
    try:
        os.chdir(ROOT)
        # remove existing folder
        print "rm -rf ./build/release"
        subprocess.check_output(["rm","-rf","./build/release"])
        print "...ok"
        print "creating directories"
        for item in ["conf","bin","core","core/web","core/web/templates"]:
            item = "./build/release/bytengine-server/" + item
            subprocess.check_output(["mkdir","-p",item])        
        subprocess.check_output(["cp","./settings/config.json","./build/release/bytengine-server/conf/config.json"])
        # setup config file directories
        config = json.load(open(os.path.join(ROOT,"settings","config.json")))
        if not os.path.exists(config["general"]["log"]):
            subprocess.check_output(["mkdir","-p",config["general"]["log"]])
        if not os.path.exists(config["web"]["upload_tmp"]):
            subprocess.check_output(["mkdir","-p",config["web"]["upload_tmp"]])
        if not os.path.exists(config["bfs"]["attachments_dir"]):
            subprocess.check_output(["mkdir","-p",config["bfs"]["attachments_dir"]])
        subprocess.check_output(["cp","-r","./web/static","./build/release/bytengine-server/core/web"])
        print "...ok"
    except subprocess.CalledProcessError, e:
        print e.output
    except Exception, err:
        print err

def buildBytengine():
    try:
        os.chdir(ROOT)
        print "building bytengine server ..."
        subprocess.check_output(["go","build"])
        print "... ok\n"
        subprocess.check_output(["mv","./bytengine","./build/release/bytengine-server/bin/bytengine"])
    except subprocess.CalledProcessError, e:
        print e.output
        sys.exit(1)
    except Exception, err:
        print err
        sys.exit(1)

def buildWebTemplates():
    print "building web templates ..."
    os.chdir(ROOT)
    _loader = FileSystemLoader(os.path.join(ROOT, "web", "templates"))
    _env = Environment(loader=_loader)
    _templatedir = "build/release/bytengine-server/core/web/templates"

    # build index.html
    _index_template_data = {
        "page_title":"Bytengine :: Home",
        "footer":False,
        "page_description":"Bytengine is an HTTP based data and binary content server \
                           that is designed to be scalable and developer friendly",
        "navbar":None
    }
    _template = _env.get_template("index.html")
    _file = open(os.path.join(ROOT,_templatedir,"index.html"),"w")
    _file.write(_template.render(_index_template_data))
    _file.close()

    # build terminal.html
    _terminal_template_data = {
        "page_title":"Bytengine :: Terminal",
        "footer":False,
        "page_description":"Bytengine web based terminal",
        "page_keywords":"terminal, web console, bash",
        "navbar":NavBar,
        "navbar_active":"Terminal"
    }
    _template = _env.get_template("terminal.html")
    _file = open(os.path.join(ROOT,_templatedir,"terminal.html"),"w")
    _file.write(_template.render(_terminal_template_data))
    _file.close()

    # get commands
    _keys = cmds.Repository.keys()
    _keys.sort()
    command_list = []
    categories = {}
    for item in _keys:
        cmd = cmds.Repository[item]
        command_list.append(cmd)
        if cmd["category"] in categories:
            categories[cmd["category"]].append(cmd["command"])
        else:
            categories[cmd["category"]] = [cmd["command"]]

    # build commands.html
    _doc_template_data = {
        "page_title":"Bytengine :: Commands",
        "footer":True,
        "navbar_active":"Commands",
        "page_description":"Bytengine Commands",
        "page_keywords":"help, man pages, guide",
        "navbar":NavBar,
        "commands":categories
    }

    _template = _env.get_template("commands.html")
    _file = open(os.path.join(ROOT,_templatedir,"commands.html"),"w")
    _file.write(_template.render(_doc_template_data))
    _file.close()
    print "...ok"

def buildDocumentationFiles():
    print "building docs..."
    os.chdir(ROOT)
    docs_dir = "./docs/"
    docs_build_dir = "./_build/json"
    docs_dist_dir = os.path.join(os.getcwd(),"build/release/bytengine-server/core/web/static/docs")

    try:
        # move to docs directory
        os.chdir(docs_dir)
        # make docs html & json
        subprocess.check_output(["make","clean"])
        subprocess.check_output(["make","-b","html"])
        subprocess.check_output(["make","-b","json"])
        # get all relevant 'fjson' files and process
        _files = [docs_build_dir + "/index.fjson"]
        for item in glob.glob(docs_build_dir + "/page_*.fjson"):
            _files.append(item)
        # create html pages
        pages = []
        pages_data = {}

        for item in _files:
            fname = os.path.basename(item).split(".")[0]
            j = json.load(open(item,"r"))
            # this is a hack for images
            body = j["body"].replace('=\"../_images/','=\"/static/docs/images/')
            
            f = open(docs_dist_dir + "/" + fname + ".html","w+")
            f.write(body)
            f.close()
            pages.append(fname)
            pages_data[fname] = {"title":j["title"],"url":fname}

        # order pages
        pages.sort()
        # build template
        os.chdir(ROOT)
        _loader = FileSystemLoader(os.path.join(os.getcwd(), "web", "templates"))
        _env = Environment(loader=_loader)
        _templatedir = "./build/release/bytengine-server/core/web/templates"

        # build documentation.html
        menu = []
        for item in pages:
            menu.append(pages_data[item])
        _docs_template_data = {
            "page_title":"Bytengine :: Documentation",
            "footer":False,
            "navbar_active":"Documentation",
            "page_description":"Bytengine Documentation",
            "page_keywords":"help, docs, guide",
            "navbar":NavBar,
            "menu":menu
        }        
        _template = _env.get_template("documentation.html")
        _file = open(os.path.join(os.getcwd(),_templatedir,"documentation.html"),"w")
        _file.write(_template.render(_docs_template_data))
        _file.close()

    except subprocess.CalledProcessError, e:
        print e.output
        sys.exit(1)
    except Exception, err:
        print err
        sys.exit(1)

def buildHelpFiles():
    print "building help..."
    os.chdir(ROOT)
    docs_dir = "./build/release/bytengine-server/core/web/static/commands"
    
    for item in cmds.Repository:
        _function = getattr(cmds, item)
        cmd_help_html = ""
        try:
            raw = _function.__doc__.format(command=cmds.Repository[item]["command"])
            tmp = core.publish_parts(raw, writer=Writer())["html_body"]
            cmd_help_html = '<br/>'.join(tmp.split('\n')[1:-2])
        except Exception, e:
            print e
            cmd_help_html = "<p>Help not generated</p>"
        f = open(os.path.join(docs_dir,cmds.Repository[item]["command"]+".html"),"w+")
        f.write(cmd_help_html)
        f.close()
    print "... ok"

    # get commands
    _keys = cmds.Repository.keys()
    _keys.sort()
    command_list = []
    categories = {}
    for item in _keys:
        cmd = cmds.Repository[item]
        command_list.append(cmd)
        if cmd["category"] in categories:
            categories[cmd["category"]].append(cmd["command"])
        else:
            categories[cmd["category"]] = [cmd["command"]]
    f = open(os.path.join(docs_dir,"all.json"),"w+")
    json.dump(categories,f)
    f.close()    

if __name__ == '__main__':    
    parser = argparse.ArgumentParser(description="Custome Make File")
    parser.add_argument(
        "-p",
        dest="package",
        action="store",
        default="all",
        choices=("all","web","engine","help"),
        help="Select which packages to recompile.")
    r = parser.parse_args()
    if r.package == "all":
        setupDirs()
        buildBytengine()
        buildHelpFiles()
        buildDocumentationFiles()
        buildWebTemplates()
    elif r.package == "web":
        buildWebTemplates()
    elif r.package == "engine":
        buildBytengine()
    elif r.package == "help":
        buildHelpFiles()
        buildDocumentationFiles()
        buildWebTemplates()