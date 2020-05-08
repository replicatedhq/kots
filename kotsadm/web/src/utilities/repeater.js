export class Repeater {
    constructor() {
        this.doNotRun = true;
    }

    start = (handlerFunc, sleepMs) => {
        this.handlerFunc = handlerFunc;
        this.sleepMs = sleepMs;
        this.doNotRun = false;
        this.repeat();
    }

    stop = () => {
        this.doNotRun = true;
    }

    repeat = () => {
        if (this.doNotRun) {
            return
        }
        this.handlerFunc().finally(() => {
            setTimeout(this.repeat, this.sleepMs);
        });
    }
}