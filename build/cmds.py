#===============================================================================
#
#
#===============================================================================

Repository = {
    "initserver": {
        "command":"server.init",
        "category":"server"
    },
    "listdb": {
        "command":"server.listdb",
        "category":"server"
    },
    "newdb": {
        "command":"server.newdb",
        "category":"server"
    },
    "dropdb": {
        "command":"server.dropdb",
        "category":"server"
    },
    "newuser": {
        "command":"server.newuser",
        "category":"server"
    },
    "listuser": {
        "command":"server.listuser",
        "category":"server"
    },
    "userinfo": {
        "command":"server.userinfo",
        "category":"server"
    },
    "dropuser": {
        "command":"server.dropuser",
        "category":"server"
    },
    "newpass": {
        "command":"server.newpass",
        "category":"server"
    },
    "sysaccess": {
        "command":"server.sysaccess",
        "category":"server"
    },
    "userdb": {
        "command":"server.userdb",
        "category":"server"
    },
    "newdir": {
        "command":"newdir",
        "category":"database"
    },
    "newfile": {
        "command":"newfile",
        "category":"database"
    },
    "listdir": {
        "command":"listdir",
        "category":"database"
    },
    "rename": {
        "command":"rename",
        "category":"database"
    },
    "move": {
        "command":"move",
        "category":"database"
    },
    "copy": {
        "command":"copy",
        "category":"database"
    },
    "delete": {
        "command":"delete",
        "category":"database"
    },
    "info": {
        "command":"info",
        "category":"database"
    },
    "makepublic": {
        "command":"makepublic",
        "category":"database"
    },
    "makeprivate": {
        "command":"makeprivate",
        "category":"database"
    },
    "readfile": {
        "command":"readfile",
        "category":"database"
    },
    "modfile": {
        "command":"modfile",
        "category":"database"
    },
    "deletebinary": {
        "command":"deletebinary",
        "category":"database"
    },
    "counter": {
        "command":"counter",
        "category":"database"
    },
    "select": {
        "command":"select",
        "category":"database"
    },
    "set": {
        "command":"set",
        "category":"database"
    },
    "unset": {
        "command":"unset",
        "category":"database"
    },
    "whoami": {
        "command":"whoami",
        "category":"database"
    }
}

# bytengine commands

def initserver():
    """
    **{command}**

    **{command}** resets Bytengine to its install time state. All user data,
    content and sessions will be deleted. This command is only executable the
    *administrator*.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command}
    """
    pass

def listdb():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [regex]

    **{command}** lists all databases the current user has access to.  If the 
    user is admin the command returns all available databases.
    The **[regex]** option filters the list of returned items

    **Return Value**::

        {{"status":"ok", "data":["db(1)",...,"db(n)"]}}

    **Example**::

        {command}

        {command} `^\w`        
    """
    pass

def newdb():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [database]

    **{command}** creates a new database. Database names can contain only alpha-
    numeric characters and must start with a letter.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "test"       
    """
    pass

def dropdb():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [database]

    **{command}** deletes an existing database. This command will delete all
    files (including any binary data) and directories.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "test"       
    """
    pass

def newuser():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username] [password]

    **{command}** creates a new system user. Both username and password must
    be composed of only alpha-numeric characters. The password has a minimum
    length of 8 characters.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "user1" "password1"
    """
    pass

def listuser():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [regex]

    **{command}** lists all system users. The **[regex]** option filters the 
    list of returned items

    **Return Value**::

        {{"status":"ok", "data":[{{"username":"xxx","active":true}},...]}}

    **Example**::

        {command}

        {command} `^\w`        
    """
    pass

def whoami():
    """
    **{command}**
    
    **Synopsis**:
    
    {command}

    **{command}** returns current session user.

    **Return Value**::

        {{
          "status":"ok",
          "data":"username"
        }}

    **Example**::

        {command}
    """
    pass


def userinfo():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username]

    **{command}** gets account metadata for the specified username. Included in
    the info is a list of all databases the user has been granted access to.

    **Return Value**::

        {{
          "status":"ok",
          "data":{{
              "username":"xxx",
              "active":false,
              "databases":["db(1)",..."db(n)"]
          }}
        }}

    **Example**::

        {command} "user1"      
    """
    pass

def dropuser():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username]

    **{command}** deletes an existing user.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "user1"       
    """
    pass

def newpass():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username] [password]

    **{command}** creates a new password for the system user. Password must
    be composed of only alpha-numeric characters. The password has a minimum
    length of 8 characters.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "user1" "password2"
    """
    pass

def sysaccess():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username] grant | deny

    **{command}** grants or denies user access to **Bytengine**.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "user1" grant
        {command} "user1" deny
    """
    pass

def userdb():
    """
    **{command}**
    
    **Synopsis**:
    
    {command} [username] [database] grant | deny

    **{command}** grants or denies user access to a **database** in **Bytengine**.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        {command} "user1" "db1" grant
        {command} "user1" "db34" deny
    """
    pass

def newdir():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** creates a directory at the given path in the *[database]*.
    Paths are unix/linux style paths (i.e. with forward slash separator) and the
    last element in the path will be the name of the directory.
    The **root directory ('/')** is created by default and cannot be modified.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images
        @mydb.{command} /user/local/my.project.dir_12-b
    """
    pass

def newfile():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [json data]

    **{command}** creates a file with the given *JSON data* as content in the
    *[database]*.
    Similarily to directories the file will be created at the given path.
    Paths are unix/linux style paths (i.e. with forward slash separator) and the
    last element in the path will be the name of the file.
    The [json data] argument must be a valid JSON object.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/index.html
        {{"title":"welcome","body":"Hello World!"}}

        @mydb.{command} /users/user1 {{"name":{{"first":"jason"}}}}
    """
    pass

def listdir():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** lists all content at the given path. The path mus poin to a
    directory.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images
        @mydb.{command} /user/local/my.project.dir_12-b
    """
    pass

def rename():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [newname]

    **{command}** renames the file or directory at the given path.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/index.html "index.php"
    """
    pass

def move():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [destination path]

    **{command}** moves the file or directory (including its contents) at the
    given path to the given destination directory. This operation checks for 
    duplicate names in the destination directory.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images /tmp/images
        @mydb.{command} /config.json /etc/bytengine/conf
    """
    pass

def copy():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [destination path]

    **{command}** copies the file or directory (including its contents) at the
    given path to the given destination path. This operation checks for 
    duplicate names in the destination directory.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images /tmp/images
        @mydb.{command} /config.json /etc/bytengine/conf/config_copy.json
    """
    pass

def delete():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** deletes the file or directory (including its contents) at the
    given path.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images
        @mydb.{command} /config.json
    """
    pass

def info():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** retrives metadata for the file or directory at the given path.

    **Return Value**::

        {{
            "status":"ok",
            "data": {{
                "attachment": {{
                    "mime":"image/jpeg",
                    "size":19201
                }},
                "created":"2013:05:15-19:56:43.9485",
                "name":"oum1",
                "parent":"/",
                "public":true,
                "type":"file"
            }}
        }}


        {{
            "status":"ok",
            "data": {{
                "content_count":2,
                "created":"2013:05:15-19:56:43.9485",
                "name":"www",
                "parent":"/var",
                "public":true,
                "type":"directory"
            }}
        }}

    **Example**::

        @mydb.{command} /oum1
        @mydb.{command} /var/tmp
    """
    pass

def makepublic():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** makes a file or directory's files available to be served publicly
    as static content by **bytengine**. When applied to a directory, all sub content
    will be made public.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images
        @mydb.{command} /images/image1.jpg
    """
    pass

def makeprivate():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** makes a file or directory's files private and only accessible
    by authorized users. When applied to a directory, all sub content will be 
    made private.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images
        @mydb.{command} /images/image1.jpg
    """
    pass

def readfile():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [field(1), field(2),...,field(n)]

    **{command}** returns the JSON data of a file at the given path. Field names
    can be included to limit the size of the dataset.

    **Return Value**::

        {{"status":"ok", "data":{{...}}}}

    **Example**::

        @mydb.{command} /var/www/website1/images/file1
        @mydb.{command} /images/image1.jpg ["author","date"]
    """
    pass

def modfile():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path] [json data]

    **{command}** overwrites a file's content with the supplied JSON data.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /var/www/website1/images/img1.jpg {{"author":"Paco"}}
    """
    pass

def deletebinary():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [path]

    **{command}** deletes the binary data section (or binary attachment) of a file.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} /images/image1.jpg
    """
    pass

def counter():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} list

    @[database].{command} [counter name] get

    @[database].{command} [counter name] incr | decr | reset [value]

    **{command}** creates a **'counter'** or global integer value for the database
    which can be incremented **'incr'**, decremented **'decr'** or reset **'reset'**.
    This command can be used to create primary keys for content.
    '**{command}** list' returns all counters and their values in the database.

    **Return Value**::

        {{"status":"ok", "data":10}}

        {{
            "status":"ok",
            "data": {{
                "users":1,
                "cars.accidents":2,
                "bodyarmour.damage":85
            }}
        }}

    **Example**::

        @mydb.{command} "users" incr 1
        @mydb.{command} "cars.accidents" decr 2
        @mydb.{command} "bodyarmour.damage" reset 0
    """
    pass

def select():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [fields] in [directories] where [conditions]
    [count] | [limit] | [distinct] | [sort]

    **{command}** retrieves fields in files from the listed directories that
    fullfill the given conditions. View tutorials for further details.

    **Return Value**::

        {{"status":"ok", "data":[{{...}},...,{{...}}]}}

    **Example**::

        @mydb.{command} "name.first" "name.last" "age" in /tmp/users /system/users
        where "age" > 10 or reges("name.last","i") == `^ja`
        limit 20
    """
    pass

def set():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [assignments] in [directories] where [conditions]
    
    **{command}** modifies the fields in files from the listed directories that
    fullfill the given conditions. View tutorials for further details.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} "nationality"="ghanaian" in /system/users
        where "country" == "ghana"
    """
    pass

def unset():
    """
    **{command}**
    
    **Synopsis**:
    
    @[database].{command} [assignments] in [directories] where [conditions]
    
    **{command}** deletes the fields in files from the listed directories that
    fullfill the given conditions. View tutorials for further details.

    **Return Value**::

        {{"status":"ok", "data":1}}

    **Example**::

        @mydb.{command} "gender" "age" in /system/users
        where "country" == "ghana"
    """
    pass