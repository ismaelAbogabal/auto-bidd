function bidDetail(config) {
    return {
        bidId: config.bidId,
        coverLetter: config.coverLetter || '',
        hours: config.hours || 0,
        rate: config.rate || 0,
        reasoning: config.reasoning || '',
        qaAnswers: config.qaAnswers || [],
        streaming: false,
        editing: false,
        error: '',
        message: '',

        // internal
        _raw: '',
        _metaParsed: false,

        get total() { return this.hours * this.rate; },

        startGenerate() {
            this._stream('/api/bids/' + this.bidId + '/generate');
        },

        sendRefine() {
            if (!this.message.trim() || this.streaming) return;

            var chatDiv = document.getElementById('chat-messages');
            if (chatDiv) {
                var msgEl = document.createElement('div');
                msgEl.className = 'flex justify-end';
                msgEl.innerHTML = '<div class="bg-indigo-50 rounded-lg px-3 py-2 max-w-[80%]"><p class="text-sm text-gray-800">' + this.message.replace(/</g, '&lt;') + '</p></div>';
                chatDiv.appendChild(msgEl);
                chatDiv.scrollTop = chatDiv.scrollHeight;
            }

            var msg = encodeURIComponent(this.message);
            this.message = '';
            this._stream('/api/bids/' + this.bidId + '/refine?message=' + msg);
        },

        _stream(endpoint) {
            this.streaming = true;
            this.editing = false;
            this.coverLetter = '';
            this.hours = 0;
            this.reasoning = '';
            this.qaAnswers = [];
            this.error = '';
            this._raw = '';
            this._metaParsed = false;
            window.scrollTo({ top: 0, behavior: 'smooth' });

            var self = this;
            var es = new EventSource(endpoint);

            es.addEventListener('delta', function(e) {
                self._raw += JSON.parse('"' + e.data + '"');
                self._parse();
            });

            es.addEventListener('done', function(e) {
                es.close();
                self._parse();
                self.streaming = false;
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
        },

        _parse() {
            var idx = this._raw.indexOf('---META---');
            if (idx === -1) {
                this.coverLetter = this._raw;
                return;
            }
            this.coverLetter = this._raw.substring(0, idx).trim();
            if (!this._metaParsed) {
                var metaStr = this._raw.substring(idx + 10).trim();
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
                    this.reasoning = meta.reasoning || '';
                    this.qaAnswers = meta.qa_answers || [];
                    this._metaParsed = true;
                } catch(e) {}
            }
        }
    };
}
