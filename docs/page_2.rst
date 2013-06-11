*************
Why Bytengine
*************

Bytengine's purpose is to make the storage and retrieval of content more intuitive.
Therefore, instead of dealing with key-values, objects ids and primary keys, you can access
your data  using **file paths** just as you would with a regular file system.

Bytengine's **file system** is modelled on the linux file system where you have a **root directory '/'**
and file paths separated by a **forward slash '/'**.

Bytengine stores your content in Files that can further be organised in Directories. Files
are JSON documents to which you can **attach** any binary/text data (docx, txt, pdf, jpeg, etc...).
The JSON layer of the File is what is queried using Bytengine's **@database.select** statement.

So for example if you were creating a blog post, a possible file and directory layout could be:

.. code-block:: guess

    /*------- new database ------*/

    server.newdb "wordpress";

    /*------- create blog directories ------*/

    @wordpress.newdir /blog;
    @wordpress.newdir /blog/posts;

    /*------- new blog post layout ------*/
    
    @wordpress.newdir /blog/posts/post1;
    @wordpress.newfile /blog/posts/post1/index {
        "title":"Hello World",
        "body":"Blah blah blah..."
    };
    
    /*------- create blog post assets directory ------*/

    @wordpress.newdir /blog/posts/post1/pics;

    /*------- create asset file ------*/
    /*------- the file extension will be used as the HTTP Content-Type header -------*/

    @wordpress.newfile /blog/posts/post1/pics/pic1.jpg {"title":"Picture of Me!"};

    /*------- make asset public so it can be served via HTTP GET ------*/

    @wordpress.makepublic /blog/posts/post1/pics/pic1.jpg;

Save the above script as 'ch2_script.bsl' for example and the do the following:

First login and get a **session id**.

.. code-block:: python

    >>> import json
    >>> import requests
    >>> url = "http://localhost:8500/bfs/prq/login"
    >>> data = {"username":"user1","password":"password"}
    >>> r = requests.post(url, data=data)
    >>> j = json.loads(r.text)
    >>> sessionid = j["data"]

Load and execute the Bytengine script

.. code-block:: python

    >>> f = open("ch2_script.bsl","r")
    >>> _script = f.read()
    >>> url = "http://localhost:8500/bfs/prq/run"
    >>> data = {"ticket":sessionid,"script":_script}
    >>> r = requests.post(url, data=data)

Request an asset upload ticket:

.. code-block:: python

    >>> upload_request_url = "http://localhost:8500/bfs/prq/upload"
    >>> bytengine_file_path = "/blog/posts/post1/pics/pic1.jpg"
    >>> data = {"db":"test","ticket":ticket,"path":bytengine_file_path}
    >>> r = requests.post(upload_request_url, data=data)
    >>> j = json.loads(r.text)
    >>> uploadticket = j["data"]

Upload file from our hard drive to Bytengine:

.. code-block:: python

    >>> upload_url = "http://localhost:8500/bfs/upload/" + uploadticket
    >>> local_file_path = "/home/me/Pictures/picture1.jpg"
    >>> files = {'file':open(local_file_path, 'rb')}
    >>> r = requests.post(upload_url, files=files)
    >>> j = json.loads(r.text)
    >>> print j["status"]
    ok

Accessing the image from your html page would be as follows:

.. code-block:: html

    <html>
        <head></head>
        <body>
            <img src="http://localhost:8500/cds/fa/wordpress/blog/posts/post1/pics/pic1.jpg" />
        </body>
    </html>
