import * as _ from "lodash";
import * as fs from "fs";
import * as path from "path";
import { describe, it } from "mocha";
import { expect } from "chai";
import { parseBackupLogs, BackupLogLine } from "./parseBackupLogs";
import { parse } from "logfmt";

describe("Velero backup logs", () => {
  /*
   * The logs file for this test has 3 separate execs, beginning on lines 15, 18, and 21. The execs
   * on lines 15 and 21 are indistiguishable because they have the same command running in the same
   * container. This test ensures our parser returns 3 separate exec results and not 2.
   */
  describe("indistinguishable execs", () => {
    it("should parse separately", () => {
      const logs = fs.readFileSync(path.join(__dirname, "test/indistinguishable-exec.txt"));
      const parsed = parseBackupLogs(logs);

      expect(parsed).to.deep.equal({
        errors: [],
        warnings: [],
        execs:
         [ { namespace: 'test',
             podName: 'example-nginx-7968bff768-qwttn',
             containerName: 'nginx',
             command: '[/bin/bash -c echo hello]',
             hookName: 'echo-hook',
             phase: 'pre',
             started: '2019-12-24T20:41:40Z',
             stdout: "hello\n",
             finished: '2019-12-24T20:41:40Z',
             stderr: '' },
           { namespace: 'test',
             podName: 'example-nginx-7968bff768-qwttn',
             containerName: 'nginx',
             command: '[/bin/bash -c echo $(date) > timestamp]',
             hookName: 'echo-hook',
             phase: 'pre',
             started: '2019-12-24T20:41:40Z',
             stdout: '',
             finished: '2019-12-24T20:41:40Z',
             stderr: '' },
           { namespace: 'test',
             podName: 'example-nginx-7968bff768-qwttn',
             containerName: 'nginx',
             command: '[/bin/bash -c echo hello]',
             phase: 'pre',
             hookName: 'echo-hook',
             started: '2019-12-24T20:41:40Z',
             stdout: "hello\n",
             finished: '2019-12-24T20:41:41Z',
             stderr: '' } ]
      });
    });
  });

  describe("errors", () => {
    it("should return errors", () => {
      const logs = fs.readFileSync(path.join(__dirname, "test/fail.txt"));
      const parsed = parseBackupLogs(logs);

      expect(parsed).to.deep.equal({
				errors:
				[ { title: 'Error executing hook',
					message: 'no such container: "api"' },
				{ title: 'Error backing up item',
					message: 'no such container: "api"' } ],
				warnings: [],
				execs:
				[ { namespace: 'default',
					podName: 'api-6d4c67c975-jc8gp',
					containerName: 'api',
					command: '[/bin/bash -c sleep 6]',
					hookName: 'hook-1',
					phase: 'pre',
					started: '2019-12-21T02:50:54Z',
					stdout: '',
					finished: '2019-12-21T02:51:00Z',
					stderr: '' },
				{ namespace: 'default',
					podName: 'api-6d4c67c975-jc8gp',
					containerName: 'api',
					command: '[/bin/bash -c sleep 6]',
					hookName: 'hook-1',
					phase: 'post',
					started: '2019-12-21T02:51:00Z',
					stdout: '',
					finished: '2019-12-21T02:51:07Z',
					stderr: '' } ]
			});
		});
  });
});
