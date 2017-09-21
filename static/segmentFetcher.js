class SegmentFetcher {
    constructor(manifestUrl, sourceUrl, sourceBuffer) {
        this.manifestUrl = manifestUrl;
        this.sourceUrl = sourceUrl;
        this.sourceBuffer = sourceBuffer;

        this.reset();
    }

    reset() {
        this.currentIndex = -1;
        this.segments = null;
        this.fetchQueue = [];
        this.halted = false;
    }

    fetchManifest(url) {
        return new Promise(function(resolve, reject) {
            var xhr = new XMLHttpRequest;
            xhr.open('get', url);
            xhr.onload = function () {
                resolve(JSON.parse(xhr.response));
            };
            xhr.send();
        });
    }

    fetchBuffer (index, start, end) {
        return new Promise((resolve, reject) => {
            var xhr = new XMLHttpRequest;
            xhr.open('get', this.sourceUrl);
            xhr.responseType = 'arraybuffer';
            xhr.setRequestHeader("Range", "bytes=" + start + "-" + end);
            xhr.onload = function () {
                xhr.response.index = index;
                resolve(xhr.response);
            };
            xhr.send();
        });
    };

    appendBuffer(buffer) {
        return new Promise((resolve, reject) => {
            this.sourceBuffer.addEventListener('updateend', function () {
                resolve();
            });

            try {
                this.sourceBuffer.appendBuffer(buffer);
            }
            catch (e) {
                console.warn(e);

                this.halt();
                this.currentIndex = buffer.index
                reject(e)
            }
        });
    }

    fetchAndAppend(index) {
        return new Promise((resolve, reject) => {
            let startRange, endRange;
            if (index == -1) {
                startRange = 0;
                endRange = this.segments[0].Start - 1;
            }
            else {
                startRange = this.segments[index].Start;
                endRange = this.segments[index].End;
            }

            this.fetchBuffer(index, startRange, endRange).then(this.appendBuffer.bind(this)).then(() =>{
                resolve();
            });
        });
    }

    isHalted() {
        return this.halted;
    }

    unhalt() {
        this.halted = false;

        this.queueNextSegmentFetch();
        this.process();
    }

    halt() {
        this.halted = true;
    }

    queueNextSegmentFetch() {
        console.log('curr index', this.currentIndex);
        this.fetchQueue.push(this.fetchAndAppend.bind(this, this.currentIndex));
        this.currentIndex++;
    }

    process() {
        let fetchAndAppendSegment = this.fetchQueue.shift();
        if (!fetchAndAppendSegment) {
            console.warn('Fetch queue empty');
            return;
        }

        fetchAndAppendSegment().then(() => {
            if (!this.halted) {
                this.queueNextSegmentFetch();
                this.process();
            }
        });
    }

    init(startIndex=0, endIndex=3) {
        return new Promise((resolve, reject) => {
            this.fetchManifest(this.manifestUrl).then((segments) => {
                this.segments = segments

                // Init segment load
                this.queueNextSegmentFetch()

                this.currentIndex = startIndex;
                for (var i = this.currentIndex; i < endIndex; i++) {
                    this.queueNextSegmentFetch()
                }

                resolve();
            });
        });
    }
}