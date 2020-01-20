import * as _ from "lodash";
import { SnapshotError, SnapshotHook, SnapshotHookPhase } from "../snapshot";
import { parse } from "logfmt";

const stdoutPrefix = "stdout: ";
const stderrPrefix = "stderr: ";

export interface ParsedBackupLogs {
  execs: Array<SnapshotHook>;
  errors: Array<SnapshotError>;
  warnings: Array<SnapshotError>;
}

export interface BackupLogLine {
  time: string,
  level: string,
  msg: string,
  name?: string, // pod name for exec logs
  namespace?: string,
  hookName?: string,
  hookContainer?: string,
  hookPhase?: SnapshotHookPhase,
  hookCommand?: string,
  error?: string,
  warning?: string,
};

function execKey(line: any): string {
  return `${line.namespace}/${line.name}/${line.hookContainer}/${line.hookName}/${line.hookPhase}/${line.hookCommand}`;
}

function isExecBegin(line: any): boolean {
  return line.msg === "running exec hook";
}

function isExecStdout(line: any): boolean {
  return _.startsWith(line.msg, stdoutPrefix);
}

function isExecStderr(line: any): boolean {
  return _.startsWith(line.msg, stderrPrefix);
}

function isError(line: any): boolean {
  return line.level === "error";
}

function isWarning(line: any): boolean {
  return line.level === "warninng";
}

function isExec(line: any): boolean {
  return !!line.hookName;
}

/*
 * This routine groups log statements by execs. The problem is that if two execs are defined for the
 * same container with the same command there is no way to distinguish them from the content of the
 * log line i.e. the execKey function returns the same value. Hence this routine is stateful.
 */
export function parseBackupLogs(buffer: Buffer): ParsedBackupLogs {
  const logs = _.map(buffer.toString().split("\n"), (s) => {
    // "\\n" (2 char) => "\n") (1 char) because the parser assumes every escape sequence is for a printable
    // https://github.com/csquared/node-logfmt/blob/e279d43cde019acfcca0abe99ea6b7ce8327e1f7/lib/logfmt_parser.js#L46
    return parse(s.replace("\\n", "\n")) as BackupLogLine;
  });
  const errors: Array<SnapshotError>  = [];
  const warnings: Array<SnapshotError> = [];
  const execs: Array<SnapshotHook> = [];
  const openExecs: any = {};

  _.each(logs, (line) => {
    if (isExecBegin(line)) {
      const key = execKey(line);
      const open = openExecs[key];
      if (open) {
        // close out the existing exec with the same key
        execs.push(open);
        delete openExecs[key];
      }

      const exec = {
        namespace: line.namespace,
        podName: line.name,
        containerName: line.hookContainer,
        command: line.hookCommand,
        hookName: line.hookName,
        phase: line.hookPhase,
        started: line.time,
      };
      openExecs[key] = exec;

      return;
    }

    if (isExecStdout(line)) {
      const open = openExecs[execKey(line)];
      if (!open) {
        console.log("Dropping stdout from backup logs");
        return;
      }
      open.stdout = line.msg.replace(/^stdout: /, "");
      open.finished = line.time;
      return;
    }

    if (isExecStderr(line)) {
      const open = openExecs[execKey(line)];
      if (!open) {
        console.log("Dropping stderr from backup logs");
        return;
      }
      open.stderr = line.msg.replace(/^stderr: /, "");
      open.finished = line.time;
      return;
    }

    if (isError(line) && isExec(line)) {
      const open = openExecs[execKey(line)];
      if (!open) {
        console.log("Dropping exec error from backup logs");
        return;
      }
      open.errors.push({ title: line.msg, message: line.error });
      return;
    }

    if (isWarning(line) && isExec(line)) {
      const open = openExecs[execKey(line)];
      if (!open) {
        console.log("Dropping exec warning from backup logs");
        return;
      }
      open.warnings.push({ title: line.msg, message: line.error });
      return;
    }

    if (isError(line)) {
      errors.push({ title: line.msg, message: line.error! });
      return;
    }

    if (isWarning(line)) {
      warnings.push({ title: line.msg, message: line.warning! });
    }
  });

  return {
    errors,
    warnings,
    execs: execs.concat(_.values(openExecs)),
  };
}
