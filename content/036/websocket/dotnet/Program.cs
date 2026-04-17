using System.Collections.Concurrent;
using System.Net.WebSockets;
using System.Text;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var clients = new ConcurrentDictionary<Guid, WebSocket>();

app.UseWebSockets(new WebSocketOptions { KeepAliveInterval = TimeSpan.FromSeconds(30) });

app.MapGet("/health", () => "ok");

app.Map("/ws", async context =>
{
    if (!context.WebSockets.IsWebSocketRequest)
    {
        context.Response.StatusCode = 400;
        return;
    }

    using var socket = await context.WebSockets.AcceptWebSocketAsync();
    var id = Guid.NewGuid();
    clients[id] = socket;
    Console.WriteLine($"[dotnet-ws] connected total={clients.Count}");

    var buffer = new byte[4096];
    try
    {
        while (socket.State == WebSocketState.Open)
        {
            var result = await socket.ReceiveAsync(buffer, CancellationToken.None);
            if (result.MessageType == WebSocketMessageType.Close)
            {
                await socket.CloseAsync(WebSocketCloseStatus.NormalClosure, "bye", CancellationToken.None);
                break;
            }
            var msg = Encoding.UTF8.GetString(buffer, 0, result.Count);
            Console.WriteLine($"[dotnet-ws] received: {msg}");
            var payload = Encoding.UTF8.GetBytes(msg);
            foreach (var (_, c) in clients)
            {
                if (c.State == WebSocketState.Open)
                    await c.SendAsync(payload, WebSocketMessageType.Text, true, CancellationToken.None);
            }
        }
    }
    catch (Exception ex) { Console.WriteLine($"[dotnet-ws] error: {ex.Message}"); }
    finally
    {
        clients.TryRemove(id, out _);
        Console.WriteLine($"[dotnet-ws] disconnected total={clients.Count}");
    }
});

Console.WriteLine("[dotnet-ws] listening on :8003");
app.Run("http://0.0.0.0:8003");
