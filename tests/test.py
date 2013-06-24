import requests
import json
import unittest
import os

URL = "http://localhost:8500/"

def runCommand(ticket, cmd):
    data = {"ticket":ticket, "script":cmd}
    r = requests.post(URL+"bfs/prq/run", data=data)
    return json.loads(r.text)

def getUploadTicket(ticket, db, path):
    data = {"ticket":ticket, "db":db,"path":path}
    r = requests.post(URL+"bfs/prq/upload", data=data)
    return json.loads(r.text)

def downloadFile(ticket, db, path, saveto):
    data = {"ticket":ticket, "db":db,"path":path}
    r = requests.post(URL+"bfs/prq/download", data=data)
    f = open(saveto,"wb")
    f.write(r.content)
    f.close()

def login(username, password):
    data = {"username":username, "password":password}
    r = requests.post(URL+"bfs/prq/login", data=data)
    return json.loads(r.text)

def printOutput(j):
    print json.dumps(j, indent=2)

class TestServerManagement(unittest.TestCase):
    def test_1(self):
        """test server management"""
        ticket = ""
        # login admin
        r = login("admin","admin")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        # rebuild server
        r = runCommand(ticket, "server.init")
        self.assertTrue(r["status"] == "ok")

        # re-login as admin because system would have been cleared
        r = login("admin","admin")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        # run other server commands
        r = runCommand(ticket,'server.newdb "test"')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket,'server.listdb')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue("test" in r["data"])

class TestUserManagement(unittest.TestCase):
    def test_1(self):
        """test user management"""
        ticket = ""
        # login admin
        r = login("admin","admin")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]
        # create user
        r = runCommand(ticket, 'server.newuser "john" "password"')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, 'server.listuser')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 1)
        # password update
        login("john","password")
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, 'server.newpass "john" "password2"')
        self.assertTrue(r["status"] == "ok")

        r = login("john","password")
        self.assertTrue(r["status"] == "error")

        r = login("john","password2")
        self.assertTrue(r["status"] == "ok")
        # database and system access
        r = runCommand(ticket, 'server.userinfo "john"')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"]["active"])
        self.assertTrue(len(r["data"]["databases"]) == 0)

        r = runCommand(ticket, 'server.sysaccess "john" deny')
        self.assertTrue(r["status"] == "ok")
        r = runCommand(ticket, 'server.userdb "john" "test" grant')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, 'server.userinfo "john"')
        self.assertTrue(r["status"] == "ok")
        self.assertFalse(r["data"]["active"])
        self.assertTrue(len(r["data"]["databases"]) == 1)
        # delete user
        r = runCommand(ticket, 'server.listuser')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 1)

        r = runCommand(ticket, 'server.dropuser "john"')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, 'server.listuser')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 0)

        # recreate user and grant access to 'test' database for subsequent tests
        r = runCommand(ticket, 'server.newuser "john" "password"')
        self.assertTrue(r["status"] == "ok")
        r = runCommand(ticket, 'server.userdb "john" "test" grant')
        self.assertTrue(r["status"] == "ok")

class TestContentManagement(unittest.TestCase):
    def test_1(self):
        """test content management"""
        ticket = ""
        # login admin
        r = login("john","password")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        r = runCommand(ticket, '@test.newdir /var; @test.newdir /var/www;')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, '@test.newfile /var/www/index.html {}')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, '@test.modfile /var/www/index.html {"title":"welcome","body":"Hello world!"}')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, '@test.readfile /var/www/index.html ["title","body"]')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"]["title"] == "welcome")

        r = runCommand(ticket, '@test.copy /var/www/index.html /var/www/index_copy.html')
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, '@test.listdir /var/www')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]["files"]) == 2)

        r = runCommand(ticket, '@test.copy /var/www /www')      
        self.assertTrue(r["status"] == "ok")

        r = runCommand(ticket, '@test.listdir /www')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]["files"]) == 2)

        r = runCommand(ticket, '@test.readfile /www/index_copy.html ["title","body"]')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"]["title"] == "welcome")

class TestBQL(unittest.TestCase):
    def test_1(self):
        """test content search"""
        ticket = ""
        # login admin
        r = login("john","password")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        q = """
        @test.newdir /users;
        @test.newfile /users/u1 {"name":"john","age":34, "country":"ghana"};
        @test.newfile /users/u2 {"name":"jason","age":18, "country":"ghana"};
        @test.newfile /users/u3 {"name":"juliette","age":18};
        @test.newfile /users/u4 {"name":"michelle","age":21, "country":"uk"};
        @test.newfile /users/u5 {"name":"denis","age":22, "country":"russia"};
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")

        q = """
        @test.select "name" "age" in /users
        where "country" in ["ghana"]
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 2)

        q = """
        @test.select "name" "age" in /users
        where regex("name","i") == `^j\w*n$`
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 2)

        q = """
        @test.select "name" "age" in /users
        where exists("country") == true
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 4)

    def test_2(self):
        """test content assignment"""
        ticket = ""
        # login admin
        r = login("admin","admin")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        q = """
        @test.set "country"={"name":"ghana","major_cities":["kumasi","accra"]}
        in /users
        where "country" == "ghana"
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 2)

        r = runCommand(ticket, "@test.readfile /users/u1")
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"]["country"]["name"] == "ghana")

        q = """
        @test.unset "country"
        in /users
        where exists("country") == true
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 4)

        q = """
        @test.select "name" "country"
        in /users
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 5)

    def test_3(self):
        """test content assignment with increments"""
        ticket = ""
        # login admin
        r = login("admin","admin")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        # increment age
        q = """
        @test.set "age" += 1
        in /users where
        regex("name","i") == `^j\w*n$`
        """
        r = runCommand(ticket, q)        
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 2)

        q = """
        @test.select "name" "age" in /users
        where regex("name","i") == `^j\w*n$`
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        for item in r["data"]:
            if item["content"]["name"] == "john":
                self.assertTrue(item["content"]["age"]==35)
            if item["content"]["name"] == "jason":
                self.assertTrue(item["content"]["age"]==19)

        # decrement age
        q = """
        @test.set "age" -= 1
        in /users where
        regex("name","i") == `^j\w*n$`
        """
        r = runCommand(ticket, q)        
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 2)

        q = """
        @test.select "name" "age" in /users
        where regex("name","i") == `^j\w*n$`
        """
        r = runCommand(ticket, q)
        self.assertTrue(r["status"] == "ok")
        for item in r["data"]:
            if item["content"]["name"] == "john":
                self.assertTrue(item["content"]["age"]==34)
            if item["content"]["name"] == "jason":
                self.assertTrue(item["content"]["age"]==18)

class TestCounter(unittest.TestCase):
    def test_1(self):
        """test counter"""
        ticket = ""
        # login admin
        r = login("john","password")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        r = runCommand(ticket, '@test.counter "users" incr 1; @test.counter "users" decr 1')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 0)

        r = runCommand(ticket, '@test.counter "users" reset 5;')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(r["data"] == 5)

        # create additional counters
        r = runCommand(ticket, '@test.counter "users.likes" incr 1; @test.counter "car.models" incr 1')
        self.assertTrue(r["status"] == "ok")
        
        r = runCommand(ticket, '@test.counter list')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 3)

        r = runCommand(ticket, '@test.counter list `^users`')
        self.assertTrue(r["status"] == "ok")
        self.assertTrue(len(r["data"]) == 2)

class TestAttachments(unittest.TestCase):
    def test_1(self):
        """test attchments"""
        ticket = ""
        # login admin
        r = login("john","password")
        self.assertTrue(r["status"] == "ok")
        ticket = r["data"]

        path = "/oum1"
        r = runCommand(ticket, '@test.newfile ' + path + ' {"meta":"picture of oum - morocan"}')
        self.assertTrue(r["status"] == "ok")

        # get upload ticket
        r = getUploadTicket(ticket,"test",path)
        self.assertTrue(r["status"] == "ok")

        # upload file
        url = "http://localhost:8500/bfs/upload/{0}".format(r["data"])
        f = os.path.join(os.path.dirname(os.path.abspath(__file__)),"oum.jpg")
        files = {'file':open(f, 'rb')}
        r = requests.post(url, files=files)
        r = json.loads(r.text)
        self.assertTrue(r["status"] == "ok")
        
        # make public
        r = runCommand(ticket, '@test.makepublic ' + path)
        self.assertTrue(r["status"] == "ok")

        # download
        saveto = "/tmp/oum_cdn.jpg"
        downloadFile(ticket,"test",path,saveto)

if __name__ == '__main__':
    # build test suite
    testsuite = unittest.TestSuite()
    testsuite.addTest(TestServerManagement("test_1"))
    testsuite.addTest(TestUserManagement("test_1"))
    testsuite.addTest(TestContentManagement("test_1"))
    testsuite.addTest(TestBQL("test_1"))
    testsuite.addTest(TestBQL("test_2"))
    testsuite.addTest(TestBQL("test_3"))
    testsuite.addTest(TestCounter("test_1"))
    testsuite.addTest(TestAttachments("test_1"))

    # run test
    unittest.TextTestRunner(verbosity=2).run(testsuite)