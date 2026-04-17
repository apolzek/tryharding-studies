using System.Security.Cryptography;
using System.Text;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

long received = 0;
var secret = Environment.GetEnvironmentVariable("WEBHOOK_SECRET") ?? "s3cret";

static bool Verify(byte[] body, string? sig, string secret)
{
    if (string.IsNullOrEmpty(sig)) return true;
    using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(secret));
    var hash = hmac.ComputeHash(body);
    var hex = Convert.ToHexString(hash).ToLowerInvariant();
    return CryptographicOperations.FixedTimeEquals(
        Encoding.UTF8.GetBytes(hex), Encoding.UTF8.GetBytes(sig));
}

app.MapGet("/health", () => "ok");
app.MapGet("/stats", () => Results.Json(new
{
    received = Interlocked.Read(ref received),
    ts = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
}));

app.MapPost("/webhook", async (HttpRequest req) =>
{
    using var ms = new MemoryStream();
    await req.Body.CopyToAsync(ms);
    var body = ms.ToArray();
    var sig = req.Headers["X-Signature"].FirstOrDefault();
    if (!Verify(body, sig, secret))
        return Results.Json(new { error = "invalid signature" }, statusCode: 401);

    var n = Interlocked.Increment(ref received);
    Console.WriteLine($"[dotnet-wh] #{n} bytes={body.Length}");
    return Results.Json(new { status = "accepted" }, statusCode: 202);
});

Console.WriteLine("[dotnet-wh] listening on :9003");
app.Run("http://0.0.0.0:9003");
