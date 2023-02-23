
async function share_list() {
    try {
        const response = await GoHttp("GET", "http://localhost/share", null)
        const message = await response.text();
        document.getElementById('list').innerHTML = message;
    } catch (err) {
        console.error('Caught exception', err)
    }
}