
<script src="/file/jquery.min.js"></script>
<script src="/file/chat.js"></script>

<!-- live chat div -->
<div class="message" id="log">
Welcome to chat room..
</div>
<div class="genDiv">
  <input type="text" id="msg" size="70"/>
  <input type="submit" id="sendBtn" value="发送"/>
</div>

<script type="text/javascript">
var chatServerAddr = "localhost:7200";
var chatServerChannel = "test";
var userId = 1;
var userNick = "andy";

$(function() {
  //begin connect chat server
  chat_server_conn(chatServerAddr, chatServerChannel);

  $("#sendBtn").click(function() {
        chat_message_send("msg");
    });
  $('#msg').keydown(function(e){ 
        if(e.keyCode==13){ 
            chat_message_send("msg");
        } 
    });
});
</script>
