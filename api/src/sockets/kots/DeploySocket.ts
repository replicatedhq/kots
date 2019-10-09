import { IO, SocketService, SocketSession, Socket } from "@tsed/socketio";

@SocketService("")
export class KotsDeploySocketService {
  constructor(@IO private io: SocketIO.Server) {
  }

  /**
   * Triggered when a new client connects to the Namespace.
   */
  $onConnection(@Socket socket: SocketIO.Socket, @SocketSession session: SocketSession) {
    console.log("connection event")
    const clusterToken = socket.handshake.query.token;
    console.log(clusterToken);
  }

  /**
   * Triggered when a client disconnects from the Namespace.
   */
  $onDisconnect(@Socket socket: SocketIO.Socket) {
    console.log("disconnected from socket");
  }
}
