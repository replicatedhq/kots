import { integer } from "aws-sdk/clients/cloudfront";
import { bool } from "aws-sdk/clients/signer";

export class Repeater {
    private readonly handlerFunc: any;
    private readonly sleepMs: number;
    private doNotRun: boolean;

    constructor(handlerFunc, sleepMs) {
        this.handlerFunc = handlerFunc;
        this.sleepMs = sleepMs;
        this.doNotRun = true;
    }

    start = () => {
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