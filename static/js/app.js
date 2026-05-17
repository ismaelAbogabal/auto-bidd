function bidStream(endpoint, hourlyRate) {
    return {
        raw: '',
        coverLetter: '',
        hours: 0,
        rate: hourlyRate || 0,
        total: 0,
        reasoning: '',
        qaAnswers: [],
        done: false,
        error: '',
        metaParsed: false,
        init() {
            var self = this;
            var es = new EventSource(endpoint);
            es.addEventListener('delta', function(e) {
                self.raw += JSON.parse('"' + e.data + '"');
                self._parse();
            });
            es.addEventListener('done', function(e) {
                self.done = true;
                es.close();
                self._parse();
                var data = JSON.parse(e.data);
                setTimeout(function() { window.location.href = data.redirect; }, 1500);
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
        },
        _parse() {
            var idx = this.raw.indexOf('---META---');
            if (idx === -1) {
                this.coverLetter = this.raw;
                return;
            }
            this.coverLetter = this.raw.substring(0, idx).trim();
            if (!this.metaParsed) {
                var metaStr = this.raw.substring(idx + 10).trim();
                try {
                    var jsonStr = metaStr;
                    if (jsonStr.indexOf('```') !== -1) {
                        var start = jsonStr.indexOf('{');
                        var end = jsonStr.lastIndexOf('}');
                        if (start !== -1 && end !== -1) {
                            jsonStr = jsonStr.substring(start, end + 1);
                        }
                    }
                    var meta = JSON.parse(jsonStr);
                    this.hours = meta.estimated_hours || 0;
                    this.total = this.hours * this.rate;
                    this.reasoning = meta.reasoning || '';
                    this.qaAnswers = meta.qa_answers || [];
                    this.metaParsed = true;
                } catch(e) {
                    // META JSON not complete yet, wait for more data
                }
            }
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
