package poc;

import io.javalin.Javalin;
import io.javalin.http.Context;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicLong;

public class WhServer {
    private static final AtomicLong RECEIVED = new AtomicLong();
    private static final String SECRET = System.getenv().getOrDefault("WEBHOOK_SECRET", "s3cret");

    public static void main(String[] args) {
        Javalin app = Javalin.create().start("0.0.0.0", 9002);

        app.get("/health", ctx -> ctx.result("ok"));
        app.get("/stats", ctx -> ctx.json(
                java.util.Map.of(
                        "received", RECEIVED.get(),
                        "ts", System.currentTimeMillis() / 1000
                )
        ));
        app.post("/webhook", WhServer::handleWebhook);

        System.out.println("[java-wh] listening on :9002");
    }

    private static void handleWebhook(Context ctx) {
        byte[] body = ctx.bodyAsBytes();
        String sig = ctx.header("X-Signature");
        if (sig != null && !verify(body, sig)) {
            ctx.status(401).result("invalid signature");
            return;
        }
        long n = RECEIVED.incrementAndGet();
        System.out.printf("[java-wh] #%d bytes=%d%n", n, body.length);
        ctx.status(202).json(java.util.Map.of("status", "accepted"));
    }

    private static boolean verify(byte[] body, String sig) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(SECRET.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            byte[] digest = mac.doFinal(body);
            StringBuilder hex = new StringBuilder();
            for (byte b : digest) hex.append(String.format("%02x", b));
            return java.security.MessageDigest.isEqual(
                    hex.toString().getBytes(StandardCharsets.UTF_8),
                    sig.getBytes(StandardCharsets.UTF_8));
        } catch (Exception e) {
            return false;
        }
    }
}
