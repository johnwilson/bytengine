**************
Quick Tutorial
**************
This guide will give you a quick overview of a few Bytengine api calls.

This guide assumes that you have Bytengine running locally on its default ports.
The python code uses the excellent `requests module <http://docs.python-requests.org/en/latest/>`_.
If you are using the online demo server please modify the urls accordingly.

**Ping the server**

.. code-block:: python

    >>> import requests
    >>> url = "http://localhost:8500/ping"
    >>> r = requests.get(url)
    >>> print r.text
    pong!

**Get the server version**

.. code-block:: python

    >>> url = "http://localhost:8500/bfs/grq/info/version"
    >>> r = requests.get(url)
    >>> print r.text
    0.1.0

**Login and create database**

.. code-block:: python

    >>> import json
    >>> url = "http://localhost:8500/bfs/prq/login"
    >>> data = {"username":"admin","password":"admin"}
    >>> r = requests.post(url, data=data)
    >>> j = json.loads(r.text)
    >>> print j["status"]
    ok
    >>> sessionid = j["data"]
    >>> cmd = 'server.newdb "test"; server.listdb;'
    >>> url = "http://localhost:8500/bfs/prq/run"
    >>> data = {"ticket":sessionid,"script":cmd}
    >>> r = requests.post(url, data=data)
    >>> j = json.loads(r.text)
    >>> print j["status"]
    ok
    >>> print j["data"]
    ['test']

**Content creation: BSL script**: ch1_script.bsl

.. code-block:: guess

    /* ============================================
       This is a BSL comment:

       Multiple commands can be issued in a single
       script but must be separated by a ';'
       Only results from the last command will be
       returned.
    =============================================== */
    
    /*---- create directories ----*/

    @test.newdir /myapp ;
    @test.newdir /myapp/users ;

    /*---- create file with valid JSON data ----*/

    @test.newfile /myapp/users/u1 {"name":"justin","age":24} ;
    @test.newfile /myapp/users/u2 {"name":"lola","age":57} ;
    @test.newfile /myapp/users/u3 {"name":"jenny","age":33} ;
    @test.newfile /myapp/users/u4 {"name":"sam","age":16} ;

    /*---- search for users and pipe results to pagination function ----*/

    @test.select "name" "age" in /myapp/users
    where "age" < 20 or regex("name","i") == `^j`;

**Load and execute script**

.. code-block:: python

    >>> f = open("ch1_script.bsl","r")
    >>> _script = f.read()
    >>> url = "http://localhost:8500/bfs/prq/run"
    >>> data = {"ticket":sessionid,"script":_script}
    >>> r = requests.post(url, data=data)
    >>> j = json.loads(r.text)
    >>> print j["status"]
    ok
    >>> print len(j["data"])
    3