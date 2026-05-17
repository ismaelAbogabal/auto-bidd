function bidStream(endpoint) {
    return {
        text: '',
        done: false,
        error: '',
        init() {
            var self = this;
            var es = new EventSource(endpoint);
            es.addEventListener('delta', function(e) {
                self.text += JSON.parse('"' + e.data + '"');
            });
            es.addEventListener('done', function(e) {
                self.done = true;
                es.close();
                var data = JSON.parse(e.data);
                setTimeout(function() { window.location.href = data.redirect; }, 500);
            });
            es.addEventListener('error', function(e) {
                self.error = e.data || 'Connection lost';
                es.close();
            });
            es.onerror = function() {
                if (!self.done) {
                    self.error = 'Connection lost';
                    es.close();
                }
            };
        }
    };
}

function bidRefine(bidId) {
    return {
        message: '',
        streaming: false,
        streamText: '',
        error: '',
        send() {
            if (!this.message.trim() || this.streaming) return;
            this.streaming = true;
            this.streamText = '';
            this.error = '';
            var msg = encodeURIComponent(this.message);
            this.message = '';
            var self = this;
            var es = new EventSource('/api/bids/' + bidId + '/refine?message=' + msg);
            es.addEventListener('delta', function(e) {
                self.streamText += JSON.parse('"' + e.data + '"');
            });
            es.addEventListener('done', function(e) {
                es.close();
                var data = JSON.parse(e.data);
                window.location.href = data.redirect;
            });
            es.addEventListener('error', function(e) {
                self.error = e.data || 'Connection lost';
                self.streaming = false;
                es.close();
            });
            es.onerror = function() {
                if (self.streaming) {
                    self.error = 'Connection lost';
                    self.streaming = false;
                    es.close();
                }
            };
        }
    };
}
