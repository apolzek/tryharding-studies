package poc;

import io.javalin.Javalin;
import io.javalin.websocket.WsContext;

import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

public class WsServer {
    private static final Set<WsContext> CLIENTS = ConcurrentHashMap.newKeySet();

    public static void main(String[] args) {
        Javalin app = Javalin.create().start("0.0.0.0", 8002);

        app.get("/health", ctx -> ctx.result("ok"));

        app.ws("/ws", ws -> {
            ws.onConnect(ctx -> {
                CLIENTS.add(ctx);
                System.out.println("[java-ws] connected total=" + CLIENTS.size());
            });
            ws.onMessage(ctx -> {
                String msg = ctx.message();
                System.out.println("[java-ws] received: " + msg);
                for (WsContext c : CLIENTS) {
                    if (c.session.isOpen()) c.send(msg);
                }
            });
            ws.onClose(ctx -> {
                CLIENTS.remove(ctx);
                System.out.println("[java-ws] disconnected total=" + CLIENTS.size());
            });
            ws.onError(ctx -> System.err.println("[java-ws] error: " + ctx.error()));
        });

        System.out.println("[java-ws] listening on :8002");
    }
}
