var COMMAND_LIST = [];

$(function(){
    var Command = Backbone.Model.extend({
        defaults: {
            name:"",
            request:"",
            status:"",
            response:""
        }
    });

    var Authentication = Backbone.Model.extend({
        defaults: {
            ticket:"",
            username:"",
            password:""
        }
    });
    
    var CommandHistory = Backbone.Collection.extend({
        model: Command
    });
    
    //--------------------------------------------------------------------------
    //  Application Views and widgets
    //--------------------------------------------------------------------------
    
    var CommandView = Backbone.View.extend({
        tagName: "section",
        
        //cache template
        template: _.template($("#template_ctrlCommand").html()),
        
        initialize: function(){
            this.model.bind("change:status", this.render, this);            
        },
        
        render: function(){
            switch(this.model.get("status")) {
                case "error":
                    var _template = _.template($("#template_error").html());
                    this.$el.append(_template(this.model.get("response")));            
                    terminal.commandcompleted(true);
                    break;
                case "success":
                    switch(this.model.get("name")) {
                        case "help *":
                            var _template = _.template($("#template_helpall").html());
                            var _data = this.model.get("response");
                            this.$el.append(_template(_data));
                            break;
                        case "help":
                            var _template = _.template($("#template_help").html());
                            var _data = this.model.get("response");
                            this.$el.append(_template(_data));                            
                            break;
                        case "history":
                            var _template = _.template($("#template_history").html());
                            var _data = this.model.get("response");
                            this.$el.append(_template(_data));                            
                            break;
                        case "login":
                            var _template = _.template($("#template_info").html());
                            var _data = this.model.get("response");
                            this.$el.append(_template(_data));                            
                            break;
                        default:
                            var _template = _.template($("#template_command_response").html());
                            var _data = this.model.get("response");
                            this.$el.append(_template(_data));
                            break;                    
                    }
                    // delete response data
                    this.model.set("response","");
                    // complete
                    terminal.commandcompleted(false);
                    break;
                case "pending":
                    // hide sensitive login info
                    if (this.model.get("name") == "login") {
                        this.model.set("request","login...");
                    }
                    this.$el.html(this.template(this.model.toJSON()));
                    break;
                default:
                    this.$el.html(this.template(this.model.toJSON()));
                    break;
            }            
            return this;
        }
    });
    
    var TerminalView = Backbone.View.extend({
        el: $("#ctrlTerminal"),
        initialize: function(){
            this.render();
            this.codeMirrorObj = null;
            this.inputMode = "single";
        },
        render: function(event){
            this.$el.html($("#template_ctrlTerminal").html());            
            return this;
        },
        events: {
            "click #btnRunCommand": "runcommand",
            "click #btnUndockInputBox": "undockinput"
        },
        loginRequest: function(username, password, process){
            postdata = {username:username,password:password};
            $.ajax({
                type:'POST',
                url: 'http://' + location.host + '/bfs/prq/login',
                data: postdata,
                dataType:'json',
                context: this,
                success: function(data)
                {
                    if(data.status == "ok") {
                        this.model.set({
                            username:username,
                            password:password,
                            ticket:data.data
                        });
                        process(true);
                    } else {
                        process(false, data.msg);
                    }
                    
                },
                error: function(jqXHR, textStatus, errorThrown){
                    var data = JSON.parse(jqXHR.responseText);
                    process(false, data.msg);
                }
            });
        },
        helpAllRequest: function(process){
            process(COMMAND_LIST);
        },
        helpCommandRequest: function(command, process){
            $.ajax({
                type:'GET',
                url: 'http://' + location.host + '/static/commands/'+command+".html",
                dataType:'html',
                success: function(data)
                {
                    process(data);                    
                },
                error: function(jqXHR, textStatus, errorThrown){
                    process(false, "Help command not found.");
                }
            });
        },
        commandRequest: function(script, context, retrycount, process){
            if (retrycount < 0) {
                process(false, "Couldn't run script. after "+retrycount+" attempts.")                
            } else {
                postdata = {ticket:context.model.get("ticket"),script:script};
                $.ajax({
                    type:'POST',
                    url: 'http://' + location.host + '/bfs/prq/run',
                    data: postdata,
                    dataType:'json',
                    context: context,
                    success: function(data)
                    {
                        if(data.status == "ok") {                        
                            process(data);
                        } else {
                            if(data.code == 2) { // session expired
                                // login
                                var u = context.model.get("username");
                                var p = context.model.get("password");
                                context.loginRequest(u,p,function(reply,err){
                                    if (err == null) { // retry command
                                        retrycount -= 1;
                                        context.commandRequest(script, context, retrycount, process);
                                    } else {
                                        process(false, err);
                                    }
                                });
                            } else {
                                process(false, data.msg);
                            }
                        }
                        
                    },
                    error: function(jqXHR, textStatus, errorThrown){
                        var data = JSON.parse(jqXHR.responseText);
                        process(false, data.msg);
                    }
                });
            }
        },
        runcommand: function(event){
            event.preventDefault();
            // disable control to prevent command input until processing complete
            this.$("#ctrlInputBox").attr("disabled",true);
            var request_text = $.trim(this.$("#ctrlInputBox").val());
            if (request_text.length == 0) {
                // do nothing
                this.unlockInput();
                return false;
            }

            // get command name
            var command_parts = request_text.split(' ',1);
            var cmd_name = command_parts[0];
            var _data = {
                request:request_text,
                _id:commandHistory.length,
                name:cmd_name,
                status:"pending"
            };

            // create command object
            var _command = new Command(_data);         
            // add to command list
            commandHistory.push(_command);
            // create new command view and append to 'Screen'
            var _commmandView = new CommandView({model:_command});
            this.$("#ctrlScreen").append(_commmandView.render().el);
            
            // set a view object to this to avoid using 'this'
            var view = this;

            switch(cmd_name.toLowerCase()) {
                case "history":
                    var _history = commandHistory.filter(function(cmd){
                        if (cmd.get("status") == "success") {
                            if(cmd.get("name") != "history") {
                                return true;
                            }
                        }
                        return false;
                    });
                    index = commandHistory.length -1;
                    var _command = commandHistory.at(index);
                    _command.set({response:{data:_history}});
                    _command.set({status:"success"});
                    break;
                case "cls":
                    this.clearconsole();
                    index = commandHistory.length -1;
                    var _command = commandHistory.at(index);
                    _command.set({response:{data:""}});
                    _command.set({status:"success"});
                    break;
                case "login":
                    // get all command parts
                    var all_parts = request_text.split(' ');                    
                    view.loginRequest(all_parts[1], all_parts[2], function(ok, err){
                        if(ok) {
                            var _json = {
                                data:"Welcome back "+view.model.get("username")
                            };
                            index = commandHistory.length -1;
                            var _command = commandHistory.at(index);
                            _command.set({response:_json});
                            _command.set({status:"success"});
                            
                        } else {
                            var _json = {
                                title:"Authentication Error",
                                data:"Login failed. check username and password."
                            };
                            index = commandHistory.length -1;
                            var _command = commandHistory.at(index);                    
                            _command.set({response:_json});
                            _command.set({status:"error"});
                            
                        }                        
                    });
                    break;
                case "help":
                    // get all command parts
                    var all_parts = request_text.split(' ');
                    if (all_parts.length < 2) {
                        view.helpAllRequest(function(data, err){
                            if (err == null) {
                                var _json = {
                                    data:COMMAND_LIST
                                };
                                index = commandHistory.length -1;
                                var _command = commandHistory.at(index);
                                _command.set({response:_json, name:"help *"});
                                _command.set({status:"success"});
                                
                            } else {
                                var _json = {
                                    title:"Help Error",
                                    data:err
                                };
                                index = commandHistory.length -1;
                                var _command = commandHistory.at(index);                    
                                _command.set({response:_json});
                                _command.set({status:"error"});
                                
                            }
                        });
                    } else {
                        view.helpCommandRequest(all_parts[1], function(data, err){
                            if (err == null) {
                                var _json = {
                                    data:data
                                };
                                index = commandHistory.length -1;
                                var _command = commandHistory.at(index);
                                _command.set({response:_json});
                                _command.set({status:"success"});
                                
                            } else {
                                var _json = {
                                    title:"Help Error",
                                    data:err
                                };
                                index = commandHistory.length -1;
                                var _command = commandHistory.at(index);                    
                                _command.set({response:_json});
                                _command.set({status:"error"});
                                
                            }
                        });
                    }
                    break;
                default: // send to bytengine
                    view.commandRequest(request_text, view, 1, function(reply, err){
                        if(err != null) {
                            var _json = {
                                title:"Request Error",
                                data:err
                            };
                            index = commandHistory.length -1;
                            var _command = commandHistory.at(index);                    
                            _command.set({response:_json});
                            _command.set({status:"error"});
                            
                        } else {
                            index = commandHistory.length -1;
                            var _command = commandHistory.at(index);
                            var _txt = JSON.stringify(reply, null, 2);
                            var _json = {data:_txt};
                            _command.set({response:_json});
                            _command.set({status:"success"});
                            
                        }
                    });
            }            
            return false;
        },
        resetInput: function(){
            this.unlockInput();
            this.$("#ctrlInputBox").val("");
        },
        lockInput: function(){
            this.$("#ctrlInputBox").attr("disabled",true);
        },
        clearconsole: function(){
            //event.preventDefault();
            // clear screen
            this.$("#ctrlScreen").empty();
            return false;
        },
        singleLineMode: function(){
            var text = this.$("#ctrlInputBox").val();
            // build control
            var ctrl = $('<input id="ctrlInputBox" name="ctrlInputBox" type="text"/>');
            ctrl.attr("autofocus","autofocus");
            ctrl.attr("autocomplete","off");
            ctrl.css("width","100%");
            ctrl.val(text);
            // add to viewport
            $("#inputViewport").html(ctrl);
    
            // setup autocomplete
            this.$("#ctrlInputBox").typeahead({
                source: function(query,process) {
                    // if database command remove the @db prefix
                    var pattern = null;
                    var isdb_command = false;
                    var data = [];
                    var prefix = "";
                    try {
                        if (query.charAt(0) == "@") {
                            isdb_command = true;
                            var i = this.query.indexOf(".");
                            if (i > -1) {
                                pattern = new RegExp("^"+query.substring(i+1));
                                prefix = query.slice(0,i+1);
                            }
                        } else {
                            pattern = new RegExp("^"+query);
                        }
                        // get relevant list of commands
                        if (isdb_command && pattern != null) {
                            _.each(COMMAND_LIST.database,function(item){
                                var result = item.match(pattern);
                                if (result != null) {
                                    data.push(prefix+item);
                                }
                            });
                            process(data);
                        } else {
                            _.each(COMMAND_LIST.server,function(item){
                                var result = item.match(pattern);
                                if (result != null) {
                                    data.push(item);
                                }
                            });
                            process(data);
                        }
                    }
                    catch(err) {
                        process([]);
                    }
                }
            });
            this.inputMode = "single";
        },
        multiLineMode: function(){
            var text = this.$("#ctrlInputBox").val();
            // build control
            var ctrl = $('<textarea id="ctrlInputBox" name="ctrlInputBox">');
            ctrl.attr("autofocus","autofocus");
            ctrl.attr("rows","1");
            ctrl.css("width","100%");
            ctrl.css("resize","none");
            ctrl.val(text);
            // add to viewport
            $("#inputViewport").html(ctrl);
            this.inputMode = "multi";
        },
        undockinput: function(event){
            event.preventDefault();
            if (this.inputMode != "multi") {
                this.multiLineMode();
            }
            
            var commandTxt = $('#ctrlInputBox').val();
            expanded_editor.setValue(commandTxt);
            
            $('#expanded_input').on('shown', function(){
                expanded_editor.refresh();
                expanded_editor.focus();
            });
            
            $('#expanded_input').on('hide', function(){
                expanded_editor.setValue("");
                expanded_editor.refresh();
            });
            
            $('#expanded_input').modal({show:true});
        },        
        commandcompleted: function(iserror){
            // scroll to bottom
            var _height = this.$("#ctrlScreen")[0].scrollHeight;
            this.$("#ctrlScreen").animate({scrollTop: _height}, 1000);
            // unlock input
            this.$("#ctrlInputBox").attr("disabled",false);

            if(iserror)
            {
                // give focus to control
                this.$("#ctrlInputBox").focus();                
            } else {
                // clear input
                this.$("#ctrlInputBox").val('');
                // revert to single mode
                if (this.inputMode != "single") {
                    this.singleLineMode();
                }
                // give control focus
                this.$("#ctrlInputBox").focus();
            }
        }
    });
    
    // Launch Application
    var ticket = null;
    var commandHistory = new CommandHistory();
    var auth = new Authentication();
    var terminal = new TerminalView({model:auth});
    terminal.singleLineMode();
    
    // get list of commands
    $.ajax({
        type:'GET',
        url: 'http://' + location.host + '/static/commands/all.json',
        dataType:'json',
        success: function(data)
        {
            COMMAND_LIST = data;            
        }
    });
    
    // setup code mirror
    var expanded_editor = CodeMirror.fromTextArea(document.getElementById("codemirrorTextarea"), {
        lineNumbers: false,
        matchBrackets: true,
        mode: "bescript",
        smartIndent: true,
        indentWithTabs: true,
        tabSize: 2
    });
    
    // setup popup button save
    $("#btnCodeMirrorSave").click(function(){
        commandTxt = expanded_editor.getValue();
        $('#ctrlInputBox').val(commandTxt);
        $('#expanded_input').modal('hide');
        $("#ctrlInputBox").focus();
    });
});
