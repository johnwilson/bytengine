// syntax highlighting for codemirror

(function(){
    CodeMirror.defineMode("bescript", function(){
        
        var keywords = "limit where in or server distinct count".split(" ");
        
        function chain(stream, state, f) {
            state.tokenize = f;
            return f(stream, state);
        }
  
        function nextUntilUnescaped(stream, end) {
            var escaped = false, next;
            while ((next = stream.next()) != null) {
                if (next == end && !escaped)
                return false;
                escaped = !escaped && next == "\\";
            }
            return escaped;
        }
        
        function lexString(quote) {
            return function(stream, state) {
                if (!nextUntilUnescaped(stream, quote))
                state.tokenize = lexInsideScript;
                return "string";
            };
        }
        
        function lexRegex(quote) {
            return function(stream, state) {
                if (!nextUntilUnescaped(stream, quote))
                state.tokenize = lexInsideScript;
                return "string-2";
            };
        }
  
        function lexComment(stream, state) {
            while (true) {
                if (stream.skipTo("*")) {
                    stream.next(); // consume *
                    if (stream.eat("/")) {
                        state.tokenize = lexInsideScript;
                        break;
                    }
                } else {
                    stream.skipToEnd();
                    break;
                }
            }
            return "comment";   
        }
        
        function lexInsideScript(stream, state) {
            var ch = stream.next();
            
            if(ch == " " || ch == "\t"){
                return null;
            }
            else if (ch == "/") {
                if (stream.eat("*")) {
                    // lex comment
                    state.tokenize = lexComment;
                    return state.tokenize(stream, state);
                }
                // lex path
                stream.match(/^[a-zA-Z_0-9\-\.//]+/);
                return "string";
            }
            else if (ch == '"' || ch == "'"){
                return chain(stream, state, lexString(ch));
            }
            else if (ch == "`"){
                return chain(stream, state, lexRegex(ch));
            }
            else if (ch == "@") {
                stream.match(/^[a-zA-Z_0-9]+/);
                return "variable-2";
            }
            else if (ch == "0" && stream.eat(/x/i)){
                stream.eatWhile(/[\da-f]/i);
                return "number";
            }
            else if (/\d/.test(ch) || ch == "-" && stream.eat(/\d/)) {
                stream.match(/^\d*(?:\.\d*)?(?:[eE][+\-]?\d+)?/);
                return "number";
            }
            else {
                stream.eatWhile(/^[\w]/);
                var word = stream.current().toLowerCase();
                for (var i=0; i<keywords.length; i++) {
                    if (keywords[i] == word) {
                        return "keyword";
                    }
                }
                return "variable";
            }
        }
        
        return {
            startState: function(){ return {tokenize:lexInsideScript}; },
            token: function(stream, state) {
                var nextItem = state.tokenize(stream,state);
                return nextItem;
            }
        };
    });
})();