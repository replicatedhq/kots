export class Repeater {
    constructor() {
        this.doNotRun = true;
    }

    start = (handlerFunc, sleepMs) => {
        if (this.isRunning()) {
            return;
        }
        this.handlerFunc = handlerFunc;
        this.sleepMs = sleepMs;
        this.doNotRun = false;
        this.repeat();
    }

    stop = () => {
        this.doNotRun = true;
    }

    isRunning = () => {
        return !this.doNotRun;
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