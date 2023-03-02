

class SigClient {
    constructor(sigAddr, serverName) {

    }
    sendSdp() {

    }
    sendIceCandidate() {

    }

    pullAnswer() {

    }
}

class Transport {
    constructor(sigAddr, serverName) {
        localConnection = new RTCPeerConnection();
        this.peer = localConnection


        this.sigCli = new SigClient(sigAddr, serverName)

        sendChannel = localConnection.createDataChannel("keepalive");
        sendChannel.onopen = handleSendChannelStatusChange;
        sendChannel.onclose = handleSendChannelStatusChange;

        localConnection.onicecandidate = e => !e.candidate
            || remoteConnection.addIceCandidate(e.candidate)
                .catch(handleAddCandidateError);

        remoteConnection.onicecandidate = e => !e.candidate
            || localConnection.addIceCandidate(e.candidate)
                .catch(handleAddCandidateError);

        localConnection.createOffer()
            .then(offer => localConnection.setLocalDescription(offer))
            .then(() => this.sigCli.sendSdp(offer))
            .then(() => {
                while (1) {
                    var answer = this.sigCli.pullAnswer()
                    if (answer != null) {
                        localConnection.setRemoteDescription(answer)
                    }
                }
            })
    }
}

