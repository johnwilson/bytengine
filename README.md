## Bytengine

[![Bytengine](http://www.bytengine.io/static/img/logo.jpg)](http://www.bytengine.io)

**[Bytengine](http://www.bytengine.io/ "Bytengine")** is a scalable content 
repository built with Go. Its API is accessible from any Http client library so 
you can start coding in your favorite language!

**[Bytengine](http://www.bytengine.io/ "Bytengine")** stores your JSON data and 
digital assets in a pseudo hierarchical file system which you query using it's 
inbuilt SQL like language.

Some of the server's features are:

* JSON data management
* Digital assets management
* HTTP based API
* Bytengine Query language (BQL)
* Pluggable data storage backends (currently supports Mongodb, Diskv, Redis)
* Documentation **[readthedocs](https://bytengine.readthedocs.org/en/latest/) - readthedocs**
* Command line interface **[bshell](http://github.com/johnwilson/bshell/ "bshell")**

## Installation

Current Build Prerequisites:

* **[Mongodb](http://docs.mongodb.org/manual/installation/ "Mongodb")**
* **[Redis](http://redis.io/download "Redis")**

You can download Bytengine binaries for:

* **[Linux amd64](http://dl.bintray.com/johnwilson/Bytengine/bytengine-server-linux64-0.2.zip "Linux amd64")**
* **[Mac OS X 10.6/10.7 amd64](http://dl.bintray.com/johnwilson/Bytengine/bytengine-server-osx64-0.2.zip "Mac OS X 10.6/10.7 amd64")**

**Extract downloaded file, 'cd' into directory and run**:

```
    ./bytengine createadmin -u="admin" -p"yourpassword"
    ./bytengine run
```

## Development

Bytengine is developed on OS X so you should adapt the following instructions
to your Os/Distro (Only tested on OS X and Ubuntu Linux)

Current Build Prerequisites:

* [Mongodb](http://docs.mongodb.org/manual/installation/ "Mongodb")
* [Redis](http://redis.io/download "Redis")
* [Go](http://golang.org/doc/install "Go")

1. Get Bytengine `go get -d github.com/johnwilson/bytengine/app`

2. `cd $GOPATH/src/github.com/johnwilson/bytengine/app`

3. Build Bytengine `go build`

4. Running Bytengine
```
	./app createadmin -u="admin" -p"yourpassword"
	./app run
```

## Some Handy Links

[Twitter](https://twitter.com/bytengine) - Twitter
