
function testJquery(url) {
    $.get(url).done(function (d) {
        console.log(d)
    }).fail(function (e) {
        console.log('hi world')
    })
}

function on_load(){
    // window.location = "http://localhost:8888/share/"
    testJquery("http://localhost:8888/share/")
}
ah.proxy({
    //请求发起前进入
    onRequest: (config, handler) => {
        console.log(config.url)
        handler.next(config);
    },
    //请求发生错误时进入，比如超时；注意，不包括http状态码错误，如404仍然会认为请求成功
    onError: (err, handler) => {
        console.log(err.type)
        handler.next(err)
    },
    //请求成功后进入
    onResponse: (response, handler) => {
        console.log(response.response)
        handler.next(response)
    }
})