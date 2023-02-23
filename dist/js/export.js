async function hello() {
    try {
        const response = await GoHttp("GET", "http://localhost/hello/1", null)
        const message = await response.text()
        console.log(message)
        document.getElementById('output').textContent = message;
    } catch (err) {
        console.error('Caught exception', err)
    }
}

async function hello_async() {
    try {
        const response = await GoHttpAsync("GET", "http://localhost/hello/1024", null)
        const reader = response.body.getReader()
        let done = false
        let totalBytes = 0;
        let output = ''
        let strResponse = - '';
        while (!done) {
            const read = await reader.read()
            done = read && read.done
            if (read.value) {
                const bytesString = String.fromCharCode(...read.value);
                output = output + bytesString;
                totalBytes = totalBytes + read.value.length;
                console.log('Read', totalBytes, 'bytes')
            }
        }
        document.getElementById('output_async').textContent = output;
        console.log(output)
    } catch (err) {
        console.error('Caught exception', err)
    }
}