.. _tutorial:

=================
LytPages tutorial
=================

In this tutorial, we are going to show you how Bytengine can be used to take care
of our web application's data storage and management needs.

Although emphasis will be placed on Bytengine commands, we will also include some
javascript and python code snippets to demonstrate how your code would interact 
with Bytengine.

In order to run the code examples you will need a Bytengine instance which you can get by:

    * `Downloading a binary <https://github.com/johnwilson/bytengine#installation>`_
    * `Building your binary <https://github.com/johnwilson/bytengine#development>`_
    * Sending a personal account request to bytengine[at]gmail.com
    * Using one of the demo server accounts (and hope nobody overwrites your directories!)

We will be using the following client-side frameworks/libraries:

    * `Jquery <http://jquery.com/>`_
    * `Underscore.js <http://underscorejs.org/>`_
    * `CodeMirror <http://codemirror.net/>`_
    * `Jquery form plugin <http://www.malsup.com/jquery/form/>`_

And the following python modules:

    * `Flask <http://jquery.com/>`_ ``easy_install flask``
    * `Flask-Login <https://github.com/maxcountryman/flask-login/>`_ ``easy_install flask-login``
    * `Requests <http://docs.python-requests.org/en/latest/>`_ ``easy_install requests``

Synopsis
========

The web app we are going to build during the course of this tutorial is called **LytPages**.

LytPages will be a site where developers can show off their 'script-fu' using html, css
and javascript! Each user will have a single page on which to build their masterpiece. 
We will provide them with a CMS where they can edit their code and upload any page
dependant media.

Table of Contents
=================

.. toctree::
   :maxdepth: 2

   page_1
   page_2
   page_3