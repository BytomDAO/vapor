$(function(){
  // 创建公钥按钮点击事件
  $('#btnCreate').click(function(){
    console.log('创建公私钥');
    $.ajax({
      method: 'post',  //get or post
      url: window.url + '/api/create_key_pair',
      success: function(data){
        if (data.code == 200) {
          console.log(data);
          layer.msg('创建成功')
        }
        else {
          console.log(data);
          layer.msg('创建失败', data.msg)
        }
 
        //appendKey('#keyContainer');  // 创建成功后传入返回数据调用，此处示例
      },
      error: function(err){
        layer.alert('创建失败' + err);
      }
    })
  })
  
 // 创建公钥按钮点击事件
 $('#btnGetKey').click(function(){
  console.log('获取公私钥');
  $.ajax({
    method: 'get',  //get or post
    url: window.url + '/api/get_key_pair',
    success: function(res){
      console.log(res);
      layer.msg('获取key pair成功')
      var assetHtml = template($('#keypairTpl').html(), {
        items: JSON.parse(res.data)
        });
        $('.key-pair-table').html(assetHtml);
    },
    error: function(err){
      layer.alert('获取失败' + err);
    }
  })
})
   // 追加内容到页面
   function appendKey(domId) {
    $.ajax({
      method: 'get',  //get or post
      url: window.url + '/api/get_key_pair',
      success: function(res){
        console.log(res);
        layer.msg('获取key pair成功')
        var assetHtml = template($('#keypairTpl').html(), {
          items: JSON.parse(res.data)
          });
          $('.key-pair-table').html(assetHtml);
      },
      error: function(err){
        layer.alert('获取失败' + err);
      }
    })
  }

  // 创建脚本
  $('#btnCreateScript').click(function(){
    console.log('创建脚本');
    var params = {
      "account_id": $('#account_id').val()
    };
    
    $.ajax({
      method: 'post',  //get or post
      url: window.url + '/api/create_pegin_address',
      dataType: 'json',
      contentType: 'application/json',
      data: JSON.stringify(params),
      success: function(data){
        if(data.code == 200){
          console.log(data);
          layer.msg('创建成功')
        }
        else {
            console.log(data);
            layer.msg('创建失败', data.msg)
        }
      },
      error: function(err){
        layer.alert('创建失败' + err);
      }
    })
  })

  // 获取脚本
  $('#btnGetScript').click(function(){
    console.log('获取脚本');
    $.ajax({
      method: 'get',  //get or post
      url: window.url + '/api/get_pegin_address',
      success: function(res){
        console.log(res);
        layer.msg('获取key pair成功')
        var peginAddressHtml = template($('#peginAddressTpl').html(), {
          items: JSON.parse(res.data)
          });
          $('.pegin-address-table').html(peginAddressHtml);
      },
      error: function(err){
        layer.alert('获取失败' + err);
      }
    })
  })
  // 追加内容到页面
  function appendAddress(domId) {
    $.ajax({
      method: 'get',  //get or post
      url: window.url + '/api/get_pegin_address',
      success: function(res){
        console.log(res);
        layer.msg('获取key pair成功')
        var peginAddressHtml = template($('#peginAddressTpl').html(), {
          items: JSON.parse(res.data)
          });
          $('.pegin-address-table').html(peginAddressHtml);
      },
      error: function(err){
        layer.alert('获取失败' + err);
      }
    })
  }

  // 发送
  $('#btnSendToSide').click(function(e){
    e.stopPropagation();
    e.preventDefault();
    var data = $('#formToSide').serializeArray();
    var obj = {};
    $.each(data, function () {
      obj[this.name] = this.value;
    });
    $.ajax({
      method: 'post',
      dataType: 'json',
      contentType: 'application/json',
      url: window.url + '/api/claim_tx',
      data: JSON.stringify(obj),
      success: function(res){
        if(res.code == 200){
          console.log(res);
          layer.msg('cliam tx success');
        }
        else {
            console.log(data);
            layer.msg('创建失败', data.msg)
        }
      },
      error: function(err){
        layer.alert('发送交易失败' + err);
      }
    })
  })

  // 侧链到主链
  $('#btnSendToMain').click(function(e){
    e.stopPropagation();
    e.preventDefault();
    var data = $('#formToMain').serializeArray();
    var obj = {};
    $.each(data, function () {
      if (this.value == ""){
        layer.alert('字段不能为空');
      }
      if(this.name == "root_xpubs" || this.name == "xprvs"){
        obj[this.name] = this.value.split(",");
      } else {
        obj[this.name] = this.value;
      }
      
    });

    $.ajax({
      method: 'post',
      dataType: 'json',
      contentType: 'application/json',
      url: window.url + '/api/send_to_mainchain',
      data: JSON.stringify(obj),
      success: function(res){
        if(res.code){
          console.log(res);
          layer.msg('send to mainchain success');
        }
        else {
            console.log(data);
            layer.msg('创建失败', data.msg)
        }
      },
      error: function(err){
        layer.alert('发送交易失败' + err);
      }
    })
  })
});

