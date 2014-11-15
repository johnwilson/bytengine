# Bytengine

[![BQL](https://github.com/johnwilson/bytengine/raw/master/bql.png)](#bql.snippet)

## About

**[Bytengine](https://github.com/johnwilson/bytengine "Bytengine")** is a scalable content 
repository built with Go. Its API is accessible from any Http client library so 
you can start coding in your favorite language!

**[Bytengine](https://github.com/johnwilson/bytengine "Bytengine")** stores your JSON data and 
digital assets in a pseudo hierarchical file system which you query using it's 
inbuilt SQL like language.

Some of the server's features are:

* JSON data management
* Digital assets management
* HTTP based API
* Bytengine Query language (BQL)
* Pluggable data storage backends (currently supports Mongodb, Diskv, Redis)
* Command line interface **[bshell](http://github.com/johnwilson/bshell/ "bshell")**

## Installation

Current Build Prerequisites:

* **[Mongodb](http://docs.mongodb.org/manual/installation/ "Mongodb")**
* **[Redis](http://redis.io/download "Redis")**

You can download Bytengine binaries for:

* **[Linux amd64](https://github.com/johnwilson/bytengine/releases/download/v0.2/bytengine-server-linux64-0.2.zip "Linux amd64")**
* **[Mac OS X 10.6/10.7 amd64](https://github.com/johnwilson/bytengine/releases/download/v0.2/bytengine-server-osx64-0.2.zip "Mac OS X 10.6/10.7 amd64")**

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
* [Redis](http://redis.io/download/ "Redis")
* [Go](http://golang.org/doc/install/ "Go")

1. Get Bytengine `go get -d github.com/johnwilson/bytengine/bytengine`

2. `cd $GOPATH/src/github.com/johnwilson/bytengine/bytengine`

3. Build Bytengine `go build`

4. Running Bytengine
```
	./bytengine createadmin -u="admin" -p"yourpassword"
	./bytengine run
```

## Quick Tutorial

#### Using Python + [Requests](http://docs.python-requests.org/en/latest/ "Requests")

```python

    >>> import requests
    >>> url = "http://localhost:8500/bfs/token"
    >>> data = {"username":"user","password":"password"}
    >>> r = requests.post(url, data=data)
    >>> j = r.json()
    >>> print j["status"]
    ok
    >>> token = j["data"]
    >>> cmd = 'server.newdb "test"; server.listdb;'  # issue two commands
    >>> url = "http://localhost:8500/bfs/query"
    >>> data = {"token":token,"query":cmd}
    >>> r = requests.post(url, data=data)
    >>> j = r.json()
    >>> print j["status"]
    ok
    >>> print j["data"][-1]  # get last result
    [u'test']
```

#### Using Bytengine Shell **[bshell](http://github.com/johnwilson/bshell/ "bshell")**

Login:

```
    bshell run -u=user -p=password
```

Enter commands:

```
    bql> server.newdb "test"; server.listdb;
    {
      "data": [
        true,
        [
          "test"
        ]
      ],
      "status": "ok"
    }
    bql> \s lastresult().status
    ok
```

## Some handy links

[Documentation](https://bytengine.readthedocs.org/en/latest/) - Bytengine Docs

[Twitter](https://twitter.com/bytengine) - Follow Bytengine
