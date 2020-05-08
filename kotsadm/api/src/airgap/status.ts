import net from "net";
import readline from "readline";
import path from "path";
import randomstring from "randomstring";
import mkdirp from "mkdirp";

export class StatusServer {
  socketFilename: string;
  server: any;
  hasCompleted: boolean;
  socket: any;

  constructor() {}

  async start(workspace: string): Promise<void> {
    const randsock = randomstring.generate({ capitalization: "lowercase" });
    mkdirp.sync(path.join(workspace, ".socket"));
    this.socketFilename = path.join(workspace, ".socket", `${randsock}.sock`);

    let statusServer = this;
    return new Promise((resolve, reject) => {
      const server = net.createServer(function (socket) {
        statusServer.socket = socket;
      });

      statusServer.server = server;

      // we'll be using line reader instead of raw data
      // server.on('data', function (data) {
      // });

      server.on('error', function (err) {
        reject(err);
      });

      server.listen(statusServer.socketFilename, function() {
        resolve();
      })
    });
  }

  async connection(): Promise<void> {
    const statusServer = this;
    return new Promise((resolve, reject) => {
      const t = setTimeout(() => {
        reject(new Error("Timeout waiting for FFI client to connect"));
      }, 20000);
      statusServer.server.on('connection', function(socket){
        clearTimeout(t);
        statusServer.socket = socket;
        resolve();
      });
    });
  }

  async termination(handler: Function): Promise<any> {
    const statusServer = this;
    return new Promise((resolve, reject) => {
      statusServer.server.on('end', function() {
        if (!statusServer.hasCompleted) {
          reject(new Error("FFI client has disconnected too early"));
        }
      });

      var i = readline.createInterface(statusServer.socket);
      i.on('line', function (line) {
          const obj = JSON.parse(line);
          statusServer.hasCompleted = handler(resolve, reject, obj);
      });
    });
  } 
}
