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

function bidRefine(bidId, hourlyRate) {
    return {
        message: '',
        streaming: false,
        raw: '',
        coverLetter: '',
        hours: 0,
        rate: hourlyRate || 0,
        reasoning: '',
        qaAnswers: [],
        metaParsed: false,
        error: '',
        send() {
            if (!this.message.trim() || this.streaming) return;
            this.streaming = true;
            this.raw = '';
            this.coverLetter = '';
            this.hours = 0;
            this.reasoning = '';
            this.qaAnswers = [];
            this.metaParsed = false;
            this.error = '';

            // Show the sent message in chat
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
            var self = this;
            var es = new EventSource('/api/bids/' + bidId + '/refine?message=' + msg);
            es.addEventListener('delta', function(e) {
                self.raw += JSON.parse('"' + e.data + '"');
                self._parse();
            });
            es.addEventListener('done', function(e) {
                es.close();
                self._parse();
                self.streaming = false;

                // Update the cover letter on the page in-place
                var coverEl = document.querySelector('#bid-output .prose');
                if (coverEl && self.coverLetter) {
                    coverEl.textContent = self.coverLetter;
                }

                // Update pricing sidebar if meta was parsed
                if (self.metaParsed && self.hours) {
                    var pricingData = document.querySelector('#bid-output')
                        ? document.querySelector('[x-data*="hours"]')
                        : null;
                    if (pricingData && pricingData._x_dataStack) {
                        pricingData._x_dataStack[0].hours = self.hours;
                    }
                }
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
                    this.reasoning = meta.reasoning || '';
                    this.qaAnswers = meta.qa_answers || [];
                    this.metaParsed = true;
                } catch(e) {
                    // META JSON not complete yet
                }
            }
        }
    };
}
