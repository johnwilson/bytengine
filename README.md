## Bytengine

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

## Installation

Prerequisites:

* **[Mongodb](http://docs.mongodb.org/manual/installation/ "Mongodb")**
* **[Redis](http://redis.io/download "Redis")**

You can download Bytengine binaries for:

* **[Linux amd64](http://dl.bintray.com/johnwilson/Bytengine/bytengine-server-linux64-0.1.zip "Linux amd64")**
* **[Mac OS X 10.6/10.7 amd64](http://dl.bintray.com/johnwilson/Bytengine/bytengine-server-osx64-0.1.zip "Mac OS X 10.6/10.7 amd64")**

**Extract downloaded file, 'cd' into directory and run**:

`./bin/bytengine --config ./conf/config.json`

## Development

Bytengine is developed on Ubuntu 12.04 so you should adapt the following instructions
to your Os/Distro (Windows is currently not supported)

Prerequisites:

* [Mongodb](http://docs.mongodb.org/manual/installation/ "Mongodb")

* [Redis](http://redis.io/download "Redis")

* [Go](http://golang.org/doc/install "Go")

* Python (>= 2.6)

* Make sure you have 'uuidgen' (no longer required)

1. Get Bytengine `go get -d github.com/johnwilson/bytengine`

2. Install Python sphinx documentation tool `easy_install sphinx`

3. Install Python [requests](http://docs.python-requests.org/en/latest/ "requests") `easy_install requests`

4. `cd $GOPATH/src/github.com/johnwilson/bytengine`

5. Build Bytengine `python ./build/run.py`

6. Running Bytengine
```
	cd ./build/release/bytengine-server/

	./bin/bytengine --config ./conf/config.json
```

7. Running Python test script
```
	cd $GOPATH/src/github.com/johnwilson/bytengine

	python ./tests/test.py
```

## Some Handy Links

[Documentation](https://bytengine.readthedocs.org/en/latest/) - Documentation

[Twitter](https://twitter.com/bytengine) - Twitter
