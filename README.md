Bytengine
=========

[![Bytengine](http://www.bytengine.com/static/img/logo.jpg)](http://www.bytengine.com)

**[Bytengine](http://www.bytengine.com/ "Bytengine")** is a scalable content repository built with
Mongodb, Redis, Go and Python.
Its API is accessible from any Http client library so you can start coding in your favorite language!

**[Bytengine](http://www.bytengine.com/ "Bytengine")** stores your content in a pseudo hierarchical 
file system which you query using it's inbuilt SQL like language.
Some of the server's features are:

* Content storage and retrival
* HTTP based API
* Inbuilt Query language
* Documentation

Build Bytengine
---------------

1. Install Go
    * go get github.com/vmihailenco/redis
    * go get labix.org/v2/mgo
    * go get github.com/gorilla/mux
    * go get github.com/gorilla/schema
2. Make sure you have Python (>= 2.6)
    * easy_install sphinx
    * easy_install requests
3. cd to build dir and run:
    * 'python run.py'
4. cd to build/release/bytengine-server and run:
    * ./bin/bytengine --config ./conf/config.json

Some Handy Links
----------------

[Demo Server](http://www.bytengine.com/) - Demo server

[Terminal](http://terminal.bytengine.com) - Sample application 1

Get Support!
------------

[Bytengine Google Group](http://groups.google.com/group/bytengine) - Q & A
